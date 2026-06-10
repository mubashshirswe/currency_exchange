package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/lib/pq"
	"github.com/mubashshir3767/currencyExchange/internal/types"
)

type Transaction struct {
	ID                 int64                     `json:"id"`
	Number             int64                     `json:"number"`
	DeliveredNumber    int64                     `json:"delivered_number"`
	ReceivedCompanyId  int64                     `json:"received_company_id"`
	ReceivedUserId     int64                     `json:"received_user_id"`
	ReceivedIncomes    []types.ReceivedIncomes   `json:"received_incomes"`
	DeliveredOutcomes  []types.DeliveredOutcomes `json:"delivered_outcomes"`
	DeliveredCompanyId int64                     `json:"delivered_company_id"`
	DeliveredUserId    *int64                    `json:"delivered_user_id"`
	ServiceFeeAmount   int64                     `json:"service_fee_amount"`
	ServiceFeeCurrency string                    `json:"service_fee_currency"`
	ServiceFeeDetails  string                    `json:"service_fee_details"`
	Phone              string                    `json:"phone"`
	Details            string                    `json:"details"`
	Status             int64                     `json:"status"`
	Type               int64                     `json:"type"`
	CreatedAt          time.Time                 `json:"-"`
	CreatedAtFormatted string                    `json:"created_at"`
}

type TransactionStorage struct {
	db DBTX
}

func NewTransactionStorage(db DBTX) *TransactionStorage {
	return &TransactionStorage{db: db}
}

func (s *TransactionStorage) Archive(ctx context.Context, companyId int64) error {
	query := `UPDATE transactions SET status = $1 WHERE status = $2 and company_id = $3`
	rows, err := s.db.ExecContext(ctx, query, STATUS_ARCHIVED, STATUS_COMPLETED, companyId)
	if err != nil {
		return err
	}

	res, err := rows.RowsAffected()
	if err != nil {
		return err
	}

	if res == 0 {
		return fmt.Errorf("TRANSACTION NOT FOUND")
	}

	return nil
}

func (s *TransactionStorage) allocateNumber(ctx context.Context, companyID int64) (int64, error) {
	if companyID == 0 {
		return 0, fmt.Errorf("company_id is required for transaction number")
	}

	var number int64
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO transaction_company_counters (company_id, last_number)
		VALUES ($1, 1)
		ON CONFLICT (company_id) DO UPDATE
			SET last_number = transaction_company_counters.last_number + 1
		RETURNING last_number
	`, companyID).Scan(&number)
	if err != nil {
		return 0, err
	}

	return number, nil
}

func (s *TransactionStorage) Create(ctx context.Context, tr *Transaction) error {
	receivedIncomesJSON, err := json.Marshal(tr.ReceivedIncomes)
	if err != nil {
		return err
	}

	deliveredOutcomesJSON, err := json.Marshal(tr.DeliveredOutcomes)
	if err != nil {
		return err
	}

	number, err := s.allocateNumber(ctx, tr.ReceivedCompanyId)
	if err != nil {
		return err
	}
	tr.Number = number

	deliveredNumber, err := s.allocateNumber(ctx, tr.DeliveredCompanyId)
	if err != nil {
		return err
	}
	tr.DeliveredNumber = deliveredNumber

	loc, _ := time.LoadLocation("Asia/Tashkent")
	nowUz := time.Now().In(loc)

	query := `
			INSERT INTO transactions(
				number, delivered_number, service_fee_amount, service_fee_currency, service_fee_details, received_incomes, delivered_outcomes,
	 			received_company_id, delivered_company_id, received_user_id, delivered_user_id, phone, details, status, type, created_at) 
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16) RETURNING id, created_at`

	err = s.db.QueryRowContext(
		ctx,
		query,
		tr.Number,
		tr.DeliveredNumber,
		tr.ServiceFeeAmount,
		tr.ServiceFeeCurrency,
		tr.ServiceFeeDetails,
		receivedIncomesJSON,
		deliveredOutcomesJSON,
		tr.ReceivedCompanyId,
		tr.DeliveredCompanyId,
		tr.ReceivedUserId,
		tr.DeliveredUserId,
		tr.Phone,
		tr.Details,
		STATUS_CREATED,
		tr.Type,
		nowUz,
	).Scan(
		&tr.ID,
		&tr.CreatedAt,
	)

	if err != nil {
		return err
	}

	return nil
}
func (s *TransactionStorage) Update(ctx context.Context, tr *Transaction) error {
	receivedIncomesJSON, err := json.Marshal(tr.ReceivedIncomes)
	if err != nil {
		return err
	}

	deliveredOutcomesJSON, err := json.Marshal(tr.DeliveredOutcomes)
	if err != nil {
		return err
	}

	query := `	
		UPDATE transactions SET
			service_fee_amount = $1,
			service_fee_currency = $2,
			service_fee_details = $3,
			received_incomes = $4,
			delivered_outcomes = $5,
			received_company_id = $6,
			delivered_company_id = $7,
			received_user_id = $8,
			delivered_user_id = $9,
			phone = $10,
			details = $11,
			status = $12,
			type = $13
		WHERE id = $14 AND status = $15
	`

	result, err := s.db.ExecContext(
		ctx,
		query,
		tr.ServiceFeeAmount,
		tr.ServiceFeeCurrency,
		tr.ServiceFeeDetails,
		receivedIncomesJSON,
		deliveredOutcomesJSON,
		tr.ReceivedCompanyId,
		tr.DeliveredCompanyId,
		tr.ReceivedUserId,
		tr.DeliveredUserId,
		tr.Phone,
		tr.Details,
		tr.Status,
		tr.Type,
		tr.ID,
		STATUS_CREATED,
	)

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return errors.New("transaction to update not found")
	}

	return nil
}

func (s *TransactionStorage) GetById(ctx context.Context, id int64) (*Transaction, error) {
	query := `
				SELECT id, number, delivered_number, service_fee_amount, service_fee_currency, service_fee_details, received_incomes, delivered_outcomes,
	 			received_company_id, delivered_company_id, received_user_id, delivered_user_id, phone, details, status, type, created_at
				FROM transactions WHERE id = $1 AND status = $2 ORDER BY created_at DESC
			`

	tr := &Transaction{}
	var receivedIncomesJSON []byte
	var deliveredOutcomesJSON []byte
	var serviceFeeDetails sql.NullString

	err := s.db.QueryRowContext(
		ctx,
		query,
		id,
		STATUS_CREATED,
	).Scan(
		&tr.ID,
		&tr.Number,
		&tr.DeliveredNumber,
		&tr.ServiceFeeAmount,
		&tr.ServiceFeeCurrency,
		&serviceFeeDetails,
		&receivedIncomesJSON,
		&deliveredOutcomesJSON,
		&tr.ReceivedCompanyId,
		&tr.DeliveredCompanyId,
		&tr.ReceivedUserId,
		&tr.DeliveredUserId,
		&tr.Phone,
		&tr.Details,
		&tr.Status,
		&tr.Type,
		&tr.CreatedAt)

	if err != nil {
		return nil, err
	}

	tr.ServiceFeeDetails = serviceFeeDetails.String

	if err := json.Unmarshal(receivedIncomesJSON, &tr.ReceivedIncomes); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(deliveredOutcomesJSON, &tr.DeliveredOutcomes); err != nil {
		return nil, err
	}

	loc, _ := time.LoadLocation("Asia/Tashkent")
	createdAtInTashkent := tr.CreatedAt.In(loc)
	tr.CreatedAtFormatted = createdAtInTashkent.Format("2006-01-02 15:04:05")

	return tr, nil
}

func (s *TransactionStorage) Archived(ctx context.Context, pagination types.Pagination) ([]Transaction, error) {
	query := `
				SELECT id, number, delivered_number, service_fee_amount, service_fee_currency, service_fee_details, received_incomes, delivered_outcomes,
	 			received_company_id, delivered_company_id, received_user_id, delivered_user_id, phone, details, status, type, created_at
				FROM transactions WHERE status = $1   ORDER BY created_at DESC ` + fmt.Sprintf("OFFSET %v LIMIT %v", pagination.Offset, pagination.Limit)

	rows, err := s.db.QueryContext(
		ctx,
		query,
		STATUS_ARCHIVED,
	)

	return s.ConvertRowsToObject(rows, err)
}

func (s *TransactionStorage) GetByField(
	ctx context.Context,
	search *string,
	fieldName string,
	fieldValue any,
	pagination types.Pagination,
) ([]Transaction, error) {

	if search != nil {
		log.Printf("COME search param %v", *search)
	} else {
		log.Println("DO NOT COME search param")
	}

	allowedFields := map[string]bool{
		"id":                   true,
		"received_user_id":     true,
		"delivered_user_id":    true,
		"received_company_id":  true,
		"delivered_company_id": true,
	}

	if !allowedFields[fieldName] {
		return nil, fmt.Errorf("invalid field name")
	}

	args := []any{fieldValue, STATUS_ARCHIVED}
	argIndex := 3 // ✅ TO‘G‘RI

	query := `
		SELECT id, number, delivered_number, service_fee_amount, service_fee_currency, service_fee_details, received_incomes, delivered_outcomes,
		received_company_id, delivered_company_id, received_user_id, delivered_user_id,
		phone, details, status, type, created_at
		FROM transactions
		WHERE ` + fieldName + ` = $1 AND status != $2
	`

	if search != nil && *search != "" {
		query += fmt.Sprintf(`
			AND (
				details ILIKE $%d 
				OR phone ILIKE $%d
				OR CAST(number AS TEXT) ILIKE $%d
				OR CAST(delivered_number AS TEXT) ILIKE $%d
				OR service_fee_details ILIKE $%d
			)
		`, argIndex, argIndex+1, argIndex+2, argIndex+3, argIndex+4)

		searchValue := "%" + *search + "%"

		args = append(args, searchValue, searchValue, searchValue, searchValue, searchValue)
		argIndex += 5

		if num, err := strconv.ParseInt(*search, 10, 64); err == nil {
			query += fmt.Sprintf(` OR number = $%d OR delivered_number = $%d`, argIndex, argIndex+1)
			args = append(args, num, num)
			argIndex += 2
		}
	}

	query += fmt.Sprintf(`
		ORDER BY created_at DESC
		OFFSET $%d LIMIT $%d
	`, argIndex, argIndex+1)

	args = append(args, pagination.Offset, pagination.Limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	return s.ConvertRowsToObject(rows, err)
}

func (s *TransactionStorage) GetInfos(ctx context.Context, companyId int64) ([]Transaction, error) {
	query := `
				SELECT id, number, delivered_number, service_fee_amount, service_fee_currency, service_fee_details, received_incomes, delivered_outcomes,
	 			received_company_id, delivered_company_id, received_user_id, delivered_user_id, phone, details, status, type, created_at
				FROM transactions WHERE delivered_company_id = $1 AND status = $2
			`
	rows, err := s.db.QueryContext(
		ctx,
		query,
		companyId,
		STATUS_CREATED,
	)

	return s.ConvertRowsToObject(rows, err)
}

func (s *TransactionStorage) GetByFieldAndDate(ctx context.Context, fieldName, from, to string, fieldValue any, pagination types.Pagination) ([]Transaction, error) {
	query := `
				SELECT id, number, delivered_number, service_fee_amount, service_fee_currency, service_fee_details, received_incomes, delivered_outcomes,
	 			received_company_id, delivered_company_id, received_user_id, delivered_user_id, phone, details, status, type, created_at
				FROM transactions WHERE ` + fmt.Sprintf("%v", fieldName) + ` = $1 AND created_at BETWEEN $2 AND $3 AND status != $4  ` + fmt.Sprintf("ORDER BY created_at DESC OFFSET %v LIMIT %v", pagination.Offset, pagination.Limit)

	rows, err := s.db.QueryContext(
		ctx,
		query,
		fieldValue,
		from,
		to,
		STATUS_ARCHIVED,
	)

	return s.ConvertRowsToObject(rows, err)
}

func (s *TransactionStorage) Delete(ctx context.Context, id *int64) error {
	query := `DELETE FROM transactions WHERE id = $1`

	rows, err := s.db.ExecContext(
		ctx,
		query,
		id,
	)

	if err != nil {
		return err
	}

	res, err := rows.RowsAffected()
	if err != nil {
		return err
	}

	if res == 0 {
		return errors.New("TRANSACTION NOT FOUND")
	}

	return nil
}

func (s *TransactionStorage) ConvertRowsToObject(rows *sql.Rows, err error) ([]Transaction, error) {
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var transactions []Transaction
	var receivedIncomesJSON []byte
	var deliveredOutcomesJSON []byte

	for rows.Next() {
		tr := &Transaction{}
		var serviceFeeDetails sql.NullString
		err := rows.Scan(
			&tr.ID,
			&tr.Number,
			&tr.DeliveredNumber,
			&tr.ServiceFeeAmount,
			&tr.ServiceFeeCurrency,
			&serviceFeeDetails,
			&receivedIncomesJSON,
			&deliveredOutcomesJSON,
			&tr.ReceivedCompanyId,
			&tr.DeliveredCompanyId,
			&tr.ReceivedUserId,
			&tr.DeliveredUserId,
			&tr.Phone,
			&tr.Details,
			&tr.Status,
			&tr.Type,
			&tr.CreatedAt,
		)

		if err != nil {
			return nil, err
		}

		tr.ServiceFeeDetails = serviceFeeDetails.String

		if err := json.Unmarshal(receivedIncomesJSON, &tr.ReceivedIncomes); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(deliveredOutcomesJSON, &tr.DeliveredOutcomes); err != nil {
			return nil, err
		}

		loc, _ := time.LoadLocation("Asia/Tashkent")
		createdAtInTashkent := tr.CreatedAt.In(loc)
		tr.CreatedAtFormatted = createdAtInTashkent.Format("2006-01-02 15:04:05")

		transactions = append(transactions, *tr)
	}

	return transactions, nil
}

type CompanyAmount struct {
	CompanyName         string
	Currency            string
	OlinganAmount       float64
	BerilganAmount      float64
	Remain              float64
	ServiceFeeAmount    float64
	ServiceFeeRemaining float64
}

func (s *TransactionStorage) GetCompanyFinalAmounts(ctx context.Context, companyIDs []int64, date string) ([]CompanyAmount, error) {
	query := `
with all_outcomes as (
    -- delivered_outcomes
    select 
        t.delivered_company_id as company_id,
        t.type,
        elem->>'delivered_currency' as currency,
        (elem->>'delivered_amount')::numeric as delivered_amount,
        0::numeric as received_amount,
        t.created_at
    from transactions t
    cross join jsonb_array_elements(t.delivered_outcomes) as elem
    where t.delivered_company_id = ANY($1) or t.received_company_id = ANY($1)

    union all

    -- received_incomes
    select 
        t.received_company_id as company_id,
        t.type,
        elem->>'received_currency' as currency,
        0::numeric as delivered_amount,
        (elem->>'received_amount')::numeric as received_amount,
        t.created_at
    from transactions t
    cross join jsonb_array_elements(t.received_incomes) as elem
    where t.delivered_company_id = ANY($1) or t.received_company_id = ANY($1)
)
select 
    c.name as company_name,
    a.currency,
    
    -- Olingan summalar (faqat date filter, Toshkent vaqti bo'yicha)
    coalesce(sum(
        case 
            when (a.created_at AT TIME ZONE 'Asia/Tashkent')::date = $2::date then
                case 
                    when a.type = 1 then a.delivered_amount
                    when a.type = 2 then a.received_amount
                    else 0
                end
            else 0
        end
    ),0) as olingan_amount,

    -- Berilgan summalar (faqat date filter, Toshkent vaqti bo'yicha)
    coalesce(sum(
        case 
            when (a.created_at AT TIME ZONE 'Asia/Tashkent')::date = $2::date then
                case 
                    when a.type = 1 then a.received_amount
                    when a.type = 2 then a.delivered_amount
                    else 0
                end
            else 0
        end
    ),0) as berilgan_amount,

    -- Qolgan summasi (barcha transactionlar)
    coalesce(sum(
        case 
            when a.type = 1 then a.delivered_amount
            when a.type = 2 then a.received_amount
            else 0
        end
    ),0)
    -
    coalesce(sum(
        case 
            when a.type = 1 then a.received_amount
            when a.type = 2 then a.delivered_amount
            else 0
        end
    ),0) as remain,

    -- Kunlik xizmat haqi (yakunlangan -> delivered, kutilayotgan -> received)
    coalesce((
        select sum(
            case
                when t.delivered_user_id is not null and t.delivered_company_id = a.company_id
                    then t.service_fee_amount
                when t.delivered_user_id is null and t.received_company_id = a.company_id
                    then t.service_fee_amount
                else 0
            end
        )::float
        from transactions t
        where t.service_fee_amount > 0
          and t.status != 3
          and upper(coalesce(nullif(trim(t.service_fee_currency), ''), 'SUM')) = upper(a.currency)
          and (t.created_at AT TIME ZONE 'Asia/Tashkent')::date = $2::date
          and (
              t.delivered_company_id = ANY($1)
              or t.received_company_id = ANY($1)
          )
    ), 0) as service_fee_amount

from all_outcomes a
join companies c on c.id = a.company_id
group by a.company_id, c.name, a.currency
order by a.company_id, a.currency;
`

	rows, err := s.db.QueryContext(ctx, query, pq.Array(companyIDs), date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []CompanyAmount
	for rows.Next() {
		var ca CompanyAmount
		if err := rows.Scan(
			&ca.CompanyName,
			&ca.Currency,
			&ca.OlinganAmount,
			&ca.BerilganAmount,
			&ca.Remain,
			&ca.ServiceFeeAmount,
		); err != nil {
			return nil, err
		}
		results = append(results, ca)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
