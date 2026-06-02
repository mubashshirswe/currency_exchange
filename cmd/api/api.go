package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/mubashshir3767/currencyExchange/internal/service"
	"github.com/mubashshir3767/currencyExchange/internal/store"
	"github.com/mubashshir3767/currencyExchange/internal/store/cache"
	"github.com/mubashshir3767/currencyExchange/internal/types"
)

type application struct {
	Pagination types.Pagination
	config     config
	store      store.Storage
	service    service.Service
	cacheStore cache.Storage
	dedup      *idempotencyGuard
}

type config struct {
	addr        string
	db          dbConfig
	redisConfig redisConfig
	env         string
}

type dbConfig struct {
	addr         string
	maxOpenConns int
	maxIdleConns int
	maxIdleTime  string
}

type redisConfig struct {
	addr    string
	pw      string
	db      int
	enabled bool
}

func (app *application) mount() *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Logger)
	// r.Use(chi.MiddlewareFunc(http.StripPrefix))

	r.Use(middleware.Timeout(60 * time.Second))

	r.Route("/api/v1", func(r chi.Router) {

		r.Post("/company", app.CreateCompanyHandler)

		r.Post("/users/register", app.CreateUserHandler)
		r.Post("/users/login", app.LoginUserHandler)

		r.With(app.JWTUserMiddleware()).Route("/user", func(r chi.Router) {

			r.Get("/all", app.GetAllUserHandler)
			r.Put("/{id}", app.UpdateUserHandler)
			r.Delete("/{id}", app.DeleteUserHandler)

			// Kompaniya balansi (joriy foydalanuvchi kompaniyasi bo'yicha)
			r.Get("/company-balances", app.GetMyCompanyBalancesHandler)
			r.Get("/company-balance-records", app.GetMyCompanyBalanceRecordsHandler)
			r.Post("/company-balance-records", app.CreateMyCompanyBalanceRecordHandler)
			r.Put("/company-balance-records/{id}", app.UpdateMyCompanyBalanceRecordHandler)
			r.Delete("/company-balance-records/{id}", app.DeleteMyCompanyBalanceRecordHandler)

			// Soft balance — biznes egasining daromadi (mustaqil)
			r.Get("/soft-balances", app.GetMySoftBalancesHandler)
			r.Get("/soft-balance-records", app.GetMySoftBalanceRecordsHandler)
			r.Post("/soft-balance-records", app.CreateMySoftBalanceRecordHandler)
			r.Delete("/soft-balance-records/{id}", app.DeleteMySoftBalanceRecordHandler)

			r.Route("/sessions", func(r chi.Router) {
				r.Post("/", app.UpsertUserSessionHandler)
				r.Get("/", app.ListUserSessionsHandler)
				r.Route("/{id}", func(r chi.Router) {
					r.Put("/", app.UpdateUserSessionHandler)
					r.Delete("/", app.DeleteUserSessionHandler)
				})
			})

			r.Route("/balances", func(r chi.Router) {
				r.Post("/", app.CreateBalanceHandler)
				r.Get("/all", app.GetAllBalanceHandler)
				r.Get("/user/{id}", app.GetBalanceByUserIdHandler)
				r.Get("/company/{id}", app.GetBalanceByCompanyIdHandler)
				r.Route("/{id}", func(r chi.Router) {
					r.Get("/", app.GetBalanceByIdHandler)
					r.Put("/", app.UpdateBalanceHandler)
					r.Delete("/", app.DeleteBalanceHandler)
				})
			})

			r.Route("/exchanges", func(r chi.Router) {
				r.With(app.DedupCreateMiddleware).Post("/", app.CreateExchangeHandler)
				r.With(app.DedupCreateMiddleware).Post("/v2", app.CreateExchangeV2Handler)
				r.Post("/filter", app.GetExchangesHandler)
				r.Post("/archive", app.ArchiveExchangesHandler)
				r.Get("/archived", app.ArchivedExchangesHandler)
				r.Route("/{id}", func(r chi.Router) {
					r.Put("/", app.UpdateExchangeHandler)
					r.Delete("/", app.DeleteExchangeHandler)
					r.Put("/v2", app.UpdateExchangeV2Handler)
					r.Delete("/v2", app.DeleteExchangeV2Handler)
				})
			})

			r.Route("/balance-records", func(r chi.Router) {
				r.Post("/", app.CreateBalanceRecordHandler)
				r.Post("/filter", app.GetBalanceRecordsHandler)
				r.Post("/archive", app.ArchiveBalanceRecordsHandler)
				r.Get("/archived", app.ArchivedBalanceRecordsHandler)
				r.Route("/{id}", func(r chi.Router) {
					r.Put("/", app.UpdateBalanceRecordHandler)
					r.Delete("/", app.DeleteBalanceRecordHandler)
				})
			})

			r.Route("/debtors", func(r chi.Router) {
				r.With(app.DedupCreateMiddleware).Post("/create", app.CreateDebtorsHandler)
				r.With(app.DedupCreateMiddleware).Post("/transaction", app.CreateDebtorTransactionHandler)
				r.With(app.DedupCreateMiddleware).Post("/create/v2", app.CreateDebtorsV2Handler)
				r.With(app.DedupCreateMiddleware).Post("/transaction/v2", app.CreateDebtorTransactionV2Handler)
				r.Get("/company/{id}", app.GetDebtorsByCompanyIdHandler)
				r.Get("/info/{id}", app.GetDebtorsTotalBalanceInfo)
				r.Delete("/{id}", app.DeleteDebtorsHandler)

				r.Route("/debts/{id}", func(r chi.Router) {
					r.Get("/", app.GetDebtsByDebtorIdHandler)
					r.Put("/", app.UpdateDebtsHandler)
					r.Delete("/", app.DeleteDebtsHandler)
					r.Put("/v2", app.UpdateDebtsV2Handler)
					r.Delete("/v2", app.DeleteDebtsV2Handler)
				})
			})

			r.Route("/transactions", func(r chi.Router) {
				r.With(app.DedupCreateMiddleware).Post("/create", app.CreateTransactionHandler)
				r.With(app.DedupCreateMiddleware).Post("/create/v2", app.CreateTransactionV2Handler)
				r.Post("/complete", app.CompleteTransactionHandler)
				r.Post("/complete/v2", app.CompleteTransactionV2Handler)
				r.Get("/show/process/{id}", app.GetTransactionsCompanyIdHandler)
				r.Post("/archive", app.ArchiveTransactionsHandler)
				r.Get("/archived", app.ArchivedTransactionsHandler)
				r.Get("/show/info/{date}", app.GetInfosByCompanyIdHandler)
				r.Post("/fetch.by.field", app.GetTransactionsByFieldHandler)
				r.Post("/fetch.by.field-and-date", app.GetTransactionsByFieldAndDateHandler)
				r.Route("/{id}", func(r chi.Router) {
					r.Put("/", app.UpdateTransactionHandler)
					r.Delete("/", app.DeleteTransactionHandler)
					r.Put("/v2", app.UpdateTransactionV2Handler)
					r.Delete("/v2", app.DeleteTransactionV2Handler)
				})
			})

			r.Route("/companies", func(r chi.Router) {
				r.Post("/", app.CreateCompanyHandler)
				r.Get("/all", app.GetAllCompanyHandler)
				r.Route("/{id}", func(r chi.Router) {
					r.Get("/", app.GetCompanyByIdHandler)
					r.Put("/", app.UpdateCompanyHandler)
					r.Delete("/", app.DeleteCompanyHandler)
					r.Get("/balances", app.GetCompanyBalancesHandler)
					r.Get("/balance-records", app.GetCompanyBalanceRecordsHandler)
					r.Get("/users/activity", app.GetCompanyUserActivityHandler)
				})
			})
		})

	})

	return r
}

func (app *application) run(mux *chi.Mux) error {
	srv := &http.Server{
		Addr:         app.config.addr,
		Handler:      mux,
		WriteTimeout: time.Second * 30,
		ReadTimeout:  time.Second * 10,
		IdleTimeout:  time.Minute,
	}

	fmt.Printf("server has been started on %v env %v", app.config.addr, app.config.env)
	return srv.ListenAndServe()
}

func getIDFromContext(r *http.Request) int64 {
	id := chi.URLParam(r, "id")
	ID, err := strconv.ParseInt(id, 10, 60)
	if err != nil {
		return 0
	}

	return ID
}

func (app *application) LoadPaginationInfo(r *http.Request, ctx context.Context) {
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}
	app.Pagination.Page = page

	orderBy := r.URL.Query().Get("order_by")
	if orderBy != "" {
		app.Pagination.OrderBy = orderBy
	} else {
		app.Pagination.OrderBy = "balance DESC"
	}

	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || limit < 1 {
		limit = 10
	}
	app.Pagination.Limit = limit

	offset := (page - 1) * limit
	app.Pagination.Offset = offset

	userID, _ := ctx.Value(UserKey).(int)
	app.Pagination.UserId = int64(userID)

	log.Printf("pagination  page: %v,  limit = %v", app.Pagination.Limit, app.Pagination.Offset)
}
