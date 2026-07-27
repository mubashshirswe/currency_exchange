package service

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/mubashshir3767/currencyExchange/internal/notify"
	"github.com/mubashshir3767/currencyExchange/internal/store"
	"github.com/mubashshir3767/currencyExchange/internal/types"
)

// CompanyOpsService — v2: exchange/transaction/debt operatsiyalari KOMPANIYA balansiga
// ta'sir qiladi (company_balances + company_balance_records). User balanslarga (balances)
// va eski balance_records'ga TEGMAYDI. Eski v1 servislar o'zgarmaydi.
type CompanyOpsService struct {
	store  store.Storage
	notify notify.DeliveredUser
}

func NewCompanyOpsService(store store.Storage, delivered notify.DeliveredUser) *CompanyOpsService {
	if delivered == nil {
		delivered = notify.NoopDeliveredUser{}
	}
	return &CompanyOpsService{store: store, notify: delivered}
}

type opLink struct {
	ExchangeId    *int64
	TransactionId *int64
	DebtId        *int64
}

// serviceFeeCompanyForUpdate — mavjud yozuv kompaniyasini saqlaydi; yangi yozuv uchun holatga qarab aniqlaydi.
func serviceFeeCompanyForUpdate(
	ctx context.Context,
	tx store.DBTX,
	tr *store.Transaction,
	actingCompanyID int64,
) (int64, error) {
	feeStorage := store.NewTransactionServiceFeeStorage(tx)
	existing, err := feeStorage.GetByTransactionID(ctx, tr.ID)
	if err == nil {
		return existing.CompanyID, nil
	}
	if err != sql.ErrNoRows {
		return 0, err
	}
	if tr.DeliveredUserId != nil {
		return serviceFeeCompanyAtComplete(tr, actingCompanyID, false), nil
	}
	return serviceFeeCompanyAtCreate(tr, actingCompanyID), nil
}

func serviceFeeCompanyAtCreate(tr *store.Transaction, actingCompanyID int64) int64 {
	if tr.ReceivedCompanyId != 0 {
		return tr.ReceivedCompanyId
	}
	return actingCompanyID
}

// serviceFeeCompanyAtComplete — yakunlashda kiritilgan xizmat haqi yetkazib beruvchi kompaniyaga yoziladi;
// yaratishda kiritilgan bo'lsa, qabul qiluvchi kompaniyada qoladi.
func serviceFeeCompanyAtComplete(tr *store.Transaction, actingCompanyID int64, hadFeeAtCreate bool) int64 {
	if hadFeeAtCreate {
		return serviceFeeCompanyAtCreate(tr, actingCompanyID)
	}
	if tr.DeliveredCompanyId != 0 {
		return tr.DeliveredCompanyId
	}
	return actingCompanyID
}

// resolveCompleteServiceFee — yakunlashda yuborilgan xizmat haqini aniqlaydi.
func resolveCompleteServiceFee(complete types.TransactionComplete) int64 {
	if complete.ServiceFeeAmount > 0 {
		return complete.ServiceFeeAmount
	}

	switch v := complete.RecievedServiceFee.(type) {
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return 0
		}
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
		var digits strings.Builder
		for _, r := range s {
			if unicode.IsDigit(r) {
				digits.WriteRune(r)
			} else if digits.Len() > 0 {
				break
			}
		}
		if digits.Len() == 0 {
			return 0
		}
		n, err := strconv.ParseInt(digits.String(), 10, 64)
		if err != nil {
			return 0
		}
		return n
	case float64:
		return int64(v)
	case int:
		return int64(v)
	case int64:
		return v
	default:
		return 0
	}
}

// applyCompanyOp — bitta kirim/chiqimni company_balances'ga qo'llaydi va
// company_balance_records'ga yozadi (hodim user_id + bog'langan operatsiya id bilan).
func applyCompanyOp(
	ctx context.Context,
	cbStorage *store.CompanyBalanceStorage,
	cbrStorage *store.CompanyBalanceRecordStorage,
	companyID, userID int64,
	currency string,
	amount, recordType int64,
	details string,
	link opLink,
) error {
	if amount <= 0 {
		return nil
	}

	cb, err := cbStorage.GetByCompanyIdAndCurrency(ctx, companyID, currency)
	if err != nil {
		if err != sql.ErrNoRows {
			return err
		}
		cb = &store.CompanyBalance{CompanyID: companyID, Currency: currency}
		if cerr := cbStorage.Create(ctx, cb); cerr != nil {
			return cerr
		}
	}

	if err := applyCompanyBalance(cb, recordType, amount); err != nil {
		return err
	}
	if err := cbStorage.Update(ctx, cb); err != nil {
		return err
	}

	rec := &store.CompanyBalanceRecord{
		CompanyID:        companyID,
		UserID:           userID,
		CompanyBalanceID: cb.ID,
		Amount:           amount,
		Currency:         currency,
		Type:             recordType,
		Details:          details,
		Status:           store.STATUS_CREATED,
		ExchangeId:       link.ExchangeId,
		TransactionId:    link.TransactionId,
		DebtId:           link.DebtId,
	}
	return cbrStorage.Create(ctx, rec)
}

func (s *CompanyOpsService) companyOf(ctx context.Context, userID int64) (int64, error) {
	u, err := s.store.Users.GetById(ctx, &userID)
	if err != nil {
		return 0, err
	}
	return u.CompanyId, nil
}

// reverseAndDeleteByLink — bog'langan operatsiya yozuvlarini company_balances'ga teskari
// qo'llaydi va o'chiradi. Update/Delete v2'da eski ta'sirni bekor qilish uchun ishlatiladi.
func reverseAndDeleteByLink(
	ctx context.Context,
	cbStorage *store.CompanyBalanceStorage,
	cbrStorage *store.CompanyBalanceRecordStorage,
	field string,
	id int64,
) error {
	recs, err := cbrStorage.ListByLink(ctx, field, id)
	if err != nil {
		return err
	}
	for _, rec := range recs {
		cb, err := cbStorage.GetByCompanyIdAndCurrency(ctx, rec.CompanyID, rec.Currency)
		if err != nil {
			return err
		}
		if err := reverseCompanyBalance(cb, rec.Type, rec.Amount); err != nil {
			return err
		}
		if err := cbStorage.Update(ctx, cb); err != nil {
			return err
		}
		if err := cbrStorage.Delete(ctx, rec.ID); err != nil {
			return err
		}
	}
	return nil
}

func isSumToUsdExchange(exchange *store.Exchange) bool {
	return strings.EqualFold(exchange.ReceivedCurrency, "USD") &&
		strings.EqualFold(exchange.SelledCurrency, "SUM")
}

func exchangeSoftProfit(exchange *store.Exchange) (int64, string, bool) {
	if !isSumToUsdExchange(exchange) || exchange.ProfitAmount <= 0 {
		return 0, "", false
	}
	cur := strings.ToUpper(strings.TrimSpace(exchange.ProfitCurrency))
	if cur != "USD" && cur != "SUM" {
		return 0, "", false
	}
	return exchange.ProfitAmount, cur, true
}

func applySoftBalanceOp(
	ctx context.Context,
	sbStorage *store.SoftBalanceStorage,
	sbrStorage *store.SoftBalanceRecordStorage,
	companyID, userID int64,
	currency string,
	amount int64,
	details string,
	exchangeID *int64,
) error {
	if amount <= 0 {
		return nil
	}

	sb, err := sbStorage.GetByCompanyIdAndCurrency(ctx, companyID, currency)
	if err != nil {
		if err != sql.ErrNoRows {
			return err
		}
		sb = &store.SoftBalance{CompanyID: companyID, Currency: currency}
		if cerr := sbStorage.Create(ctx, sb); cerr != nil {
			return cerr
		}
	}

	if err := applySoftBalance(sb, TYPE_BUY, amount); err != nil {
		return err
	}
	if err := sbStorage.Update(ctx, sb); err != nil {
		return err
	}

	rec := &store.SoftBalanceRecord{
		CompanyID:     companyID,
		UserID:        userID,
		SoftBalanceID: sb.ID,
		Amount:        amount,
		Currency:      currency,
		Type:          TYPE_BUY,
		Details:       details,
		Status:        store.STATUS_CREATED,
		ExchangeId:    exchangeID,
	}
	return sbrStorage.Create(ctx, rec)
}

func reverseAndDeleteSoftByLink(
	ctx context.Context,
	sbStorage *store.SoftBalanceStorage,
	sbrStorage *store.SoftBalanceRecordStorage,
	field string,
	id int64,
) error {
	recs, err := sbrStorage.ListByLink(ctx, field, id)
	if err != nil {
		return err
	}
	for _, rec := range recs {
		cb, err := sbStorage.GetByCompanyIdAndCurrency(ctx, rec.CompanyID, rec.Currency)
		if err != nil {
			return err
		}
		if err := reverseSoftBalance(cb, rec.Type, rec.Amount); err != nil {
			return err
		}
		if err := sbStorage.Update(ctx, cb); err != nil {
			return err
		}
		if err := sbrStorage.Delete(ctx, rec.ID); err != nil {
			return err
		}
	}
	return nil
}

func applyExchangeSoftProfit(
	ctx context.Context,
	sbStorage *store.SoftBalanceStorage,
	sbrStorage *store.SoftBalanceRecordStorage,
	exchange *store.Exchange,
) error {
	amount, currency, ok := exchangeSoftProfit(exchange)
	if !ok {
		return nil
	}
	details := fmt.Sprintf("Exchange foyda #%d", exchange.ID)
	if exchange.Details != "" && exchange.Details != "COMMENT" {
		details = exchange.Details
	}
	return applySoftBalanceOp(
		ctx, sbStorage, sbrStorage,
		exchange.CompanyID, exchange.UserId,
		currency, amount, details, &exchange.ID,
	)
}

// CreateExchangeV2 — exchange yaratadi va kompaniya balansiga ta'sir qiladi.
// received => kirim (BUY), selled => chiqim (SELL). exchange.UserId = amalni bajargan hodim.
func (s *CompanyOpsService) CreateExchangeV2(ctx context.Context, exchange *store.Exchange) error {
	companyID, err := s.companyOf(ctx, exchange.UserId)
	if err != nil {
		return err
	}

	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	exchangeStore := store.NewExchangeStorage(tx)
	cbStorage := store.NewCompanyBalanceStorage(tx)
	cbrStorage := store.NewCompanyBalanceRecordStorage(tx)
	sbStorage := store.NewSoftBalanceStorage(tx)
	sbrStorage := store.NewSoftBalanceRecordStorage(tx)

	exchange.CompanyID = companyID
	if err := exchangeStore.Create(ctx, exchange); err != nil {
		return fmt.Errorf("ERROR OCCURRED WHILE CREATING EXCHANGE %v", err)
	}

	link := opLink{ExchangeId: &exchange.ID}

	if err := applyCompanyOp(ctx, cbStorage, cbrStorage, companyID, exchange.UserId,
		exchange.ReceivedCurrency, exchange.ReceivedMoney, TYPE_BUY, exchange.Details, link); err != nil {
		return err
	}

	if err := applyCompanyOp(ctx, cbStorage, cbrStorage, companyID, exchange.UserId,
		exchange.SelledCurrency, exchange.SelledMoney, TYPE_SELL, exchange.Details, link); err != nil {
		return err
	}

	if err := applyExchangeSoftProfit(ctx, sbStorage, sbrStorage, exchange); err != nil {
		return err
	}

	return tx.Commit()
}

// UpdateExchangeV2 — exchange'ni yangilaydi: eski company balans ta'sirini bekor qiladi,
// exchange qatorini yangilaydi va yangi ta'sirni qo'llaydi. User balanslarga tegmaydi.
func (s *CompanyOpsService) UpdateExchangeV2(ctx context.Context, exchange *store.Exchange) error {
	companyID, err := s.companyOf(ctx, exchange.UserId)
	if err != nil {
		return err
	}

	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	exchangeStore := store.NewExchangeStorage(tx)
	cbStorage := store.NewCompanyBalanceStorage(tx)
	cbrStorage := store.NewCompanyBalanceRecordStorage(tx)
	sbStorage := store.NewSoftBalanceStorage(tx)
	sbrStorage := store.NewSoftBalanceRecordStorage(tx)

	if err := reverseAndDeleteByLink(ctx, cbStorage, cbrStorage, "exchange_id", exchange.ID); err != nil {
		return err
	}
	if err := reverseAndDeleteSoftByLink(ctx, sbStorage, sbrStorage, "exchange_id", exchange.ID); err != nil {
		return err
	}

	exchange.CompanyID = companyID
	if err := exchangeStore.Update(ctx, exchange); err != nil {
		return fmt.Errorf("ERROR OCCURRED WHILE UPDATING EXCHANGE %v", err)
	}

	link := opLink{ExchangeId: &exchange.ID}
	if err := applyCompanyOp(ctx, cbStorage, cbrStorage, companyID, exchange.UserId,
		exchange.ReceivedCurrency, exchange.ReceivedMoney, TYPE_BUY, exchange.Details, link); err != nil {
		return err
	}
	if err := applyCompanyOp(ctx, cbStorage, cbrStorage, companyID, exchange.UserId,
		exchange.SelledCurrency, exchange.SelledMoney, TYPE_SELL, exchange.Details, link); err != nil {
		return err
	}
	if err := applyExchangeSoftProfit(ctx, sbStorage, sbrStorage, exchange); err != nil {
		return err
	}

	return tx.Commit()
}

// DeleteExchangeV2 — exchange'ni o'chiradi va company balans ta'sirini bekor qiladi.
func (s *CompanyOpsService) DeleteExchangeV2(ctx context.Context, id int64) error {
	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	exchangeStore := store.NewExchangeStorage(tx)
	cbStorage := store.NewCompanyBalanceStorage(tx)
	cbrStorage := store.NewCompanyBalanceRecordStorage(tx)
	sbStorage := store.NewSoftBalanceStorage(tx)
	sbrStorage := store.NewSoftBalanceRecordStorage(tx)

	if err := reverseAndDeleteByLink(ctx, cbStorage, cbrStorage, "exchange_id", id); err != nil {
		return err
	}
	if err := reverseAndDeleteSoftByLink(ctx, sbStorage, sbrStorage, "exchange_id", id); err != nil {
		return err
	}
	if err := exchangeStore.Delete(ctx, id); err != nil {
		return fmt.Errorf("ERROR OCCURRED WHILE DELETING EXCHANGE %v", err)
	}

	return tx.Commit()
}

// invertOpType — SELL <-> BUY.
//
// Qarz yozuvi HAR DOIM amalga oshmagan pul harakatining teskarisi bo'ladi:
//   - create: pul chiqishi kerak edi (SELL), mijoz olishga kelmadi => qarz BUY,
//     debtor.balance musbat (biz qarzdormiz);
//   - complete: pul kirishi kerak edi (yetkazish BUY), mijoz keltirmadi => qarz SELL,
//     debtor.balance manfiy (mijoz qarzdor).
//
// Shu bilan birga kompaniya balansidagi ta'sir ham neytrallanadi — pul aslida
// qo'ldan qo'lga o'tmagan.
func invertOpType(t int64) int64 {
	if t == TYPE_SELL {
		return TYPE_BUY
	}
	return TYPE_SELL
}

// debtAmount — qarz yozuvi uchun bitta valyuta bo'yicha summa.
type debtAmount struct {
	Currency string
	Amount   int64
}

// debtAmountsFromIncomes — received_incomes'ni valyuta bo'yicha yig'ib qarz summalariga aylantiradi.
func debtAmountsFromIncomes(incomes []types.ReceivedIncomes) []debtAmount {
	amounts := make([]debtAmount, 0, len(incomes))
	for _, in := range incomes {
		amounts = appendDebtAmount(amounts, in.ReceivedCurrency, in.ReceivedAmount)
	}
	return amounts
}

// debtAmountsFromOutcomes — delivered_outcomes'ni valyuta bo'yicha yig'ib qarz summalariga aylantiradi.
func debtAmountsFromOutcomes(outcomes []types.DeliveredOutcomes) []debtAmount {
	amounts := make([]debtAmount, 0, len(outcomes))
	for _, out := range outcomes {
		amounts = appendDebtAmount(amounts, out.DeliveredCurrency, out.DeliveredAmount)
	}
	return amounts
}

func appendDebtAmount(amounts []debtAmount, currency string, amount int64) []debtAmount {
	if amount <= 0 || strings.TrimSpace(currency) == "" {
		return amounts
	}
	for i := range amounts {
		if amounts[i].Currency == currency {
			amounts[i].Amount += amount
			return amounts
		}
	}
	return append(amounts, debtAmount{Currency: currency, Amount: amount})
}

// resolveTransactionDebtor — qarzdorni topadi yoki yaratadi.
// debtor_id yuborilgan va kompaniya + valyuta mos bo'lsa o'sha ishlatiladi;
// mos kelmasa yoki topilmasa telefon + valyuta bo'yicha qidiriladi,
// u ham topilmasa yangi qarzdor ochiladi.
// Ikkinchi qaytish qiymati — qarzdor shu yerda yangi yaratilganini bildiradi.
func resolveTransactionDebtor(
	ctx context.Context,
	debtorsStorage *store.DebtorsStorage,
	companyID, actingUserID int64,
	input types.TransactionDebtInput,
	currency string,
) (*store.Debtors, bool, error) {
	fullName := strings.TrimSpace(input.FullName)
	phone := strings.TrimSpace(input.Phone)

	if input.DebtorID > 0 {
		debtor, err := debtorsStorage.GetById(ctx, input.DebtorID)
		switch {
		case err == nil:
			if debtor.CompanyID == companyID && debtor.Currency == currency {
				return debtor, false, nil
			}
			// Kompaniya yoki valyuta mos kelmadi — yangi qarzdor ochiladi,
			// ism/telefon esa ko'rsatilgan qarzdordan olinadi.
			if fullName == "" {
				fullName = debtor.FullName
			}
			if phone == "" {
				phone = debtor.Phone
			}
		case err == sql.ErrNoRows:
			// topilmadi — pastda yangi qarzdor ochiladi
		default:
			return nil, false, fmt.Errorf("failed to get debtor: %w", err)
		}
	}

	if phone != "" {
		debtor, err := debtorsStorage.GetByCompanyPhoneAndCurrency(ctx, companyID, phone, currency)
		if err == nil {
			return debtor, false, nil
		}
		if err != sql.ErrNoRows {
			return nil, false, fmt.Errorf("failed to lookup debtor by phone: %w", err)
		}
	}

	if fullName == "" {
		fullName = phone
	}

	debtor := &store.Debtors{
		FullName:  fullName,
		Balance:   0,
		Currency:  currency,
		UserID:    actingUserID,
		CompanyID: companyID,
		Phone:     phone,
	}
	if err := debtorsStorage.Create(ctx, debtor); err != nil {
		return nil, false, fmt.Errorf("failed to create debtor: %w", err)
	}
	return debtor, true, nil
}

const (
	// DEBT_STAGE_CREATE — qarz transaction yaratilayotganda (received_incomes) yozilgan.
	DEBT_STAGE_CREATE = 1
	// DEBT_STAGE_COMPLETE — qarz transaction yakunlanayotganda (delivered_outcomes) yozilgan.
	DEBT_STAGE_COMPLETE = 2
)

// reverseTransactionDebts — transaction'ga bog'langan qarz yozuvlarini bekor qiladi:
// company balans yozuvlari qaytariladi, debtor balansi tiklanadi, debts qatori o'chiriladi.
// Qaytgan ro'yxat — bekor qilingan qarzlar (qayta yaratishda qarzdorni saqlab qolish uchun).
func reverseTransactionDebts(
	ctx context.Context,
	tx store.DBTX,
	cbStorage *store.CompanyBalanceStorage,
	cbrStorage *store.CompanyBalanceRecordStorage,
	transactionID int64,
) ([]store.Debts, error) {
	debtsStorage := store.NewDebtsStorage(tx)
	debtorsStorage := store.NewDebtorsStorage(tx)

	debts, err := debtsStorage.ListByTransactionID(ctx, transactionID)
	if err != nil {
		return nil, err
	}

	for _, debt := range debts {
		if err := reverseAndDeleteByLink(ctx, cbStorage, cbrStorage, "debt_id", debt.ID); err != nil {
			return nil, err
		}

		if debt.DebtorID != 0 {
			debtor, err := debtorsStorage.GetById(ctx, debt.DebtorID)
			switch {
			case err == nil:
				amount := absInt64(debt.DebtedAmount)
				if debt.Type == types.TYPE_SELL {
					debtor.Balance += amount
				} else {
					debtor.Balance -= amount
				}
				if err := debtorsStorage.Update(ctx, debtor); err != nil {
					return nil, fmt.Errorf("failed to update debtor: %w", err)
				}
			case err == sql.ErrNoRows:
				// qarzdor o'chirilgan — faqat debt qatorini olib tashlaymiz
			default:
				return nil, fmt.Errorf("failed to get debtor: %w", err)
			}
		}

		if err := debtsStorage.Delete(ctx, debt.ID); err != nil {
			return nil, fmt.Errorf("failed to delete debt: %w", err)
		}
	}

	return debts, nil
}

// debtInputFromDebts — bekor qilingan qarzlardan qarzdor ma'lumotini tiklaydi, shunda
// transaction yangilanganda qarz o'sha qarzdorga qayta yoziladi. Bitta qarz bo'lsa
// debtor_id ham beriladi; bir nechta valyuta bo'lsa ism/telefon bo'yicha topiladi.
func debtInputFromDebts(debts []store.Debts, stage int) types.TransactionDebtInput {
	var input types.TransactionDebtInput
	matched := 0
	for _, debt := range debts {
		if debt.TransactionStage != stage {
			continue
		}
		matched++
		if matched == 1 {
			input = types.TransactionDebtInput{
				DebtorID: debt.DebtorID,
				FullName: debt.FullName,
				Phone:    debt.Phone,
				Details:  debt.Details,
			}
		}
	}
	if matched > 1 {
		input.DebtorID = 0
	}
	return input
}

// transactionDebtOp — transaction'dan qarz yozish uchun bir martalik amal tavsifi.
type transactionDebtOp struct {
	CompanyID     int64
	ActingUserID  int64
	TransactionID int64
	// Stage — qarz qaysi bosqichda yozilgani: DEBT_STAGE_CREATE yoki DEBT_STAGE_COMPLETE.
	Stage   int
	Input   types.TransactionDebtInput
	Amounts []debtAmount
	// DebtType — debts qatori va debtor balansi uchun qarz turi: amalga oshmagan
	// pul harakatiga teskari (payload'dagi debt_type yuborilsa, o'sha ustun turadi).
	DebtType int
	// BalanceType — company balansiga qo'llanadigan yo'nalish, u ham o'sha teskari
	// yo'nalish, lekin debt_type override'iga bog'liq emas: balans har doim neytrallanadi.
	BalanceType int64
	Details     string
}

// recordTransactionDebt — transaction bilan birga qarzdorlik yozuvlarini yaratadi.
// Har bir valyuta uchun alohida debtor + debt qatori bo'ladi (debtors bitta valyutali).
// Yaratilgan qarzlar transaction_id + transaction_stage orqali transaction'ga bog'lanadi,
// shunda transaction yangilanganda/o'chirilganda ular ham qaytariladi.
func recordTransactionDebt(
	ctx context.Context,
	tx store.DBTX,
	cbStorage *store.CompanyBalanceStorage,
	cbrStorage *store.CompanyBalanceRecordStorage,
	op transactionDebtOp,
) error {
	if !op.Input.Enabled() || len(op.Amounts) == 0 {
		return nil
	}

	debtType := op.Input.Type
	if debtType != types.TYPE_SELL && debtType != types.TYPE_BUY {
		debtType = op.DebtType
	}
	if debtType != types.TYPE_SELL && debtType != types.TYPE_BUY {
		return fmt.Errorf("invalid debt type: %d", debtType)
	}

	details := op.Details
	if trimmed := strings.TrimSpace(op.Input.Details); trimmed != "" {
		details = trimmed
	}

	debtorsStorage := store.NewDebtorsStorage(tx)
	debtsStorage := store.NewDebtsStorage(tx)

	for _, amount := range op.Amounts {
		debtor, created, err := resolveTransactionDebtor(
			ctx, debtorsStorage, op.CompanyID, op.ActingUserID, op.Input, amount.Currency,
		)
		if err != nil {
			return err
		}

		transactionID := op.TransactionID
		debt := &store.Debts{
			FullName:       debtor.FullName,
			DebtedCurrency: amount.Currency,
			DebtedAmount:   amount.Amount,
			ReceivedIncomes: []types.ReceivedIncomes{{
				ReceivedAmount:   amount.Amount,
				ReceivedCurrency: amount.Currency,
			}},
			UserID:           op.ActingUserID,
			CompanyID:        op.CompanyID,
			DebtorID:         debtor.ID,
			Details:          details,
			Phone:            debtor.Phone,
			IsBalanceEffect:  1,
			Type:             debtType,
			TransactionID:    &transactionID,
			TransactionStage: op.Stage,
		}

		if debtType == types.TYPE_SELL {
			debt.DebtedAmount = -amount.Amount
			if created {
				debt.State = 1
			}
		}

		if err := debtsStorage.Create(ctx, debt); err != nil {
			return fmt.Errorf("failed to create debt: %w", err)
		}

		link := opLink{DebtId: &debt.ID}
		for _, in := range debt.ReceivedIncomes {
			if err := applyCompanyOp(ctx, cbStorage, cbrStorage, op.CompanyID, op.ActingUserID,
				in.ReceivedCurrency, in.ReceivedAmount, op.BalanceType, details, link); err != nil {
				return err
			}
		}

		if debtType == types.TYPE_SELL {
			debtor.Balance -= amount.Amount
		} else {
			debtor.Balance += amount.Amount
		}
		if err := debtorsStorage.Update(ctx, debtor); err != nil {
			return fmt.Errorf("failed to update debtor: %w", err)
		}
	}

	return nil
}

// PerformTransactionV2 — transaction yaratadi; received_incomes kompaniya balansiga ta'sir qiladi.
// SELL => chiqim, BUY => kirim. actingUserID = amalni bajargan hodim (JWT).
// debtInput to'ldirilgan bo'lsa, o'sha tranzaksiya ichida qarzdorlik yozuvi ham yaratiladi.
func (s *CompanyOpsService) PerformTransactionV2(ctx context.Context, transaction *store.Transaction, actingUserID int64, debtInput types.TransactionDebtInput) error {
	companyID, err := s.companyOf(ctx, actingUserID)
	if err != nil {
		return err
	}

	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	transactionsStorage := store.NewTransactionStorage(tx)
	cbStorage := store.NewCompanyBalanceStorage(tx)
	cbrStorage := store.NewCompanyBalanceRecordStorage(tx)

	if transaction.ServiceFeeAmount > 0 {
		transaction.ServiceFeeCurrency = "SUM"
	}

	if err := transactionsStorage.Create(ctx, transaction); err != nil {
		return fmt.Errorf("ERROR OCCURRED WHILE Transactions.Create %v", err)
	}

	if err := NewServiceFeeService(s.store).SyncFromTransactionTx(
		ctx, tx, transaction, serviceFeeCompanyAtCreate(transaction, companyID),
	); err != nil {
		return fmt.Errorf("service fee sync on create: %w", err)
	}

	link := opLink{TransactionId: &transaction.ID}
	for _, tr := range transaction.ReceivedIncomes {
		if err := applyCompanyOp(ctx, cbStorage, cbrStorage, companyID, actingUserID,
			tr.ReceivedCurrency, tr.ReceivedAmount, transaction.Type, transaction.Details, link); err != nil {
			return err
		}
	}

	if debtInput.Phone == "" {
		debtInput.Phone = transaction.Phone
	}
	if err := recordTransactionDebt(ctx, tx, cbStorage, cbrStorage, transactionDebtOp{
		CompanyID:     companyID,
		ActingUserID:  actingUserID,
		TransactionID: transaction.ID,
		Stage:         DEBT_STAGE_CREATE,
		Input:         debtInput,
		Amounts:       debtAmountsFromIncomes(transaction.ReceivedIncomes),
		DebtType:      int(invertOpType(transaction.Type)),
		BalanceType:   invertOpType(transaction.Type),
		Details:       transaction.Details,
	}); err != nil {
		return fmt.Errorf("debt on transaction create: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	tid := transaction.ID
	phone := transaction.Phone
	details := transaction.Details
	deliveredCompanyID := transaction.DeliveredCompanyId
	go func() {
		ctxN, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		s.notify.NotifyNewOrderToCompany(ctxN, deliveredCompanyID, tid, phone, details)
		if transaction.DeliveredUserId != nil {
			uid := *transaction.DeliveredUserId
			s.notify.NotifyPendingDelivery(ctxN, &uid, tid, phone, details)
		}
	}()
	return nil
}

// AcceptTransactionV2 — 3 bosqichli oqimdagi oraliq bosqich: buyurtma qabul qilinadi.
// BALANSGA TEGMAYDI — pul harakati avvalgidek yaratish va topshirish bosqichlarida.
// Faqat status STATUS_ACCEPTED ga o'tadi va qabul qilgan hodim/kompaniya yoziladi.
func (s *CompanyOpsService) AcceptTransactionV2(ctx context.Context, transactionID, actingUserID, businessID int64) error {
	settings, err := s.store.BusinessSettings.GetByBusinessID(ctx, businessID)
	if err != nil {
		return fmt.Errorf("business settings: %w", err)
	}
	if !settings.IsThreeStage() {
		return fmt.Errorf(types.TRANSACTION_ACCEPT_DISABLED)
	}

	companyID, err := s.companyOf(ctx, actingUserID)
	if err != nil {
		return err
	}

	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	transactionsStorage := store.NewTransactionStorage(tx)

	tran, err := transactionsStorage.GetById(ctx, transactionID)
	if err != nil {
		return err
	}
	if tran.Status == store.STATUS_ACCEPTED {
		return fmt.Errorf(types.TRANSACTION_ALREADY_ACCEPTED)
	}

	if err := transactionsStorage.SetAccepted(ctx, tran.ID, actingUserID, companyID); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf(types.TRANSACTION_ALREADY_ACCEPTED)
		}
		return fmt.Errorf("ERROR OCCURRED WHILE transactionsStorage.SetAccepted %v", err)
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	tid := tran.ID
	phone := tran.Phone
	details := tran.Details
	receivedCompanyID := tran.ReceivedCompanyId
	go func() {
		ctxN, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		s.notify.NotifyOrderAccepted(ctxN, receivedCompanyID, tid, phone, details)
	}()
	return nil
}

// CompleteTransactionV2 — transaction yakunlaydi; delivered_outcomes kompaniya balansiga ta'sir qiladi.
// SELL transaction => yetkazib berish kirim (BUY); BUY transaction => chiqim (SELL).
// 3 bosqichli oqimda yakunlashdan oldin buyurtma qabul qilingan bo'lishi shart.
func (s *CompanyOpsService) CompleteTransactionV2(ctx context.Context, complete types.TransactionComplete, actingUserID, businessID int64) error {
	settings, err := s.store.BusinessSettings.GetByBusinessID(ctx, businessID)
	if err != nil {
		return fmt.Errorf("business settings: %w", err)
	}

	companyID, err := s.companyOf(ctx, actingUserID)
	if err != nil {
		return err
	}

	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	transactionsStorage := store.NewTransactionStorage(tx)
	cbStorage := store.NewCompanyBalanceStorage(tx)
	cbrStorage := store.NewCompanyBalanceRecordStorage(tx)

	tran, err := transactionsStorage.GetById(ctx, complete.TransactionID)
	if err != nil {
		return fmt.Errorf("ERROR OCCURRED WHILE transactionsStorage.GetById %v", err)
	}

	if settings.IsThreeStage() && tran.Status != store.STATUS_ACCEPTED {
		return fmt.Errorf(types.TRANSACTION_NOT_ACCEPTED)
	}

	feeAtComplete := resolveCompleteServiceFee(complete)
	if tran.ServiceFeeAmount <= 0 && feeAtComplete <= 0 {
		return fmt.Errorf(types.SERVICE_FEE_REQUIRED_AT_COMPLETE)
	}
	hadFeeAtCreate := tran.ServiceFeeAmount > 0

	link := opLink{TransactionId: &tran.ID}
	// SELL transaction => yetkazib berish kirim (BUY); BUY transaction => chiqim (SELL).
	deliveredType := invertOpType(tran.Type)
	for _, tr := range tran.DeliveredOutcomes {
		if err := applyCompanyOp(ctx, cbStorage, cbrStorage, companyID, actingUserID,
			tr.DeliveredCurrency, tr.DeliveredAmount, deliveredType, tran.Details, link); err != nil {
			return err
		}
	}

	debtInput := complete.DebtInput()
	if debtInput.Phone == "" {
		debtInput.Phone = tran.Phone
	}
	if err := recordTransactionDebt(ctx, tx, cbStorage, cbrStorage, transactionDebtOp{
		CompanyID:     companyID,
		ActingUserID:  actingUserID,
		TransactionID: tran.ID,
		Stage:         DEBT_STAGE_COMPLETE,
		Input:         debtInput,
		Amounts:       debtAmountsFromOutcomes(tran.DeliveredOutcomes),
		DebtType:      int(invertOpType(deliveredType)),
		BalanceType:   invertOpType(deliveredType),
		Details:       tran.Details,
	}); err != nil {
		return fmt.Errorf("debt on transaction complete: %w", err)
	}

	if tran.ServiceFeeAmount <= 0 && feeAtComplete > 0 {
		tran.ServiceFeeAmount = feeAtComplete
		tran.ServiceFeeCurrency = "SUM"
	}
	if details := strings.TrimSpace(complete.ServiceFeeDetails); details != "" {
		tran.ServiceFeeDetails = details
	}

	tran.Status = TRANSACTION_STATUS_COMPLETED
	tran.DeliveredUserId = &complete.DeliveredUserId
	if err := transactionsStorage.Update(ctx, tran); err != nil {
		return fmt.Errorf("ERROR OCCURRED WHILE transactionsStorage.Update %v", err)
	}

	if err := NewServiceFeeService(s.store).SyncFromTransactionTx(
		ctx, tx, tran, serviceFeeCompanyAtComplete(tran, companyID, hadFeeAtCreate),
	); err != nil {
		return fmt.Errorf("service fee sync on complete: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	deliveredID := complete.DeliveredUserId
	tid := tran.ID
	details := tran.Details
	go func() {
		ctxN, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		s.notify.NotifyDeliveryCompleted(ctxN, deliveredID, tid, details)
	}()
	return nil
}

// UpdateTransactionV2 — transaction'ni yangilaydi: eski company balans ta'sirini bekor qiladi,
// transaction qatorini yangilaydi va yangi ta'sirni qayta qo'llaydi (received + delivered).
// Transaction'ga bog'langan qarz yozuvlari ham bekor qilinib, yangi summalar bilan
// qayta yoziladi — qarzdor payload'da ko'rsatilmasa, eski qarzdagi qarzdor saqlanadi.
func (s *CompanyOpsService) UpdateTransactionV2(ctx context.Context, transaction *store.Transaction, actingUserID int64, debtInput types.TransactionDebtInput) error {
	companyID, err := s.companyOf(ctx, actingUserID)
	if err != nil {
		return err
	}

	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	transactionsStorage := store.NewTransactionStorage(tx)
	cbStorage := store.NewCompanyBalanceStorage(tx)
	cbrStorage := store.NewCompanyBalanceRecordStorage(tx)

	if err := reverseAndDeleteByLink(ctx, cbStorage, cbrStorage, "transaction_id", transaction.ID); err != nil {
		return err
	}

	oldDebts, err := reverseTransactionDebts(ctx, tx, cbStorage, cbrStorage, transaction.ID)
	if err != nil {
		return fmt.Errorf("debt reverse on transaction update: %w", err)
	}

	if transaction.ServiceFeeAmount > 0 {
		transaction.ServiceFeeCurrency = "SUM"
	}

	if err := transactionsStorage.Update(ctx, transaction); err != nil {
		return fmt.Errorf("ERROR OCCURRED WHILE UPDATING TRANSACTION %v", err)
	}

	link := opLink{TransactionId: &transaction.ID}

	if transaction.ReceivedUserId != 0 {
		for _, tr := range transaction.ReceivedIncomes {
			if err := applyCompanyOp(ctx, cbStorage, cbrStorage, companyID, actingUserID,
				tr.ReceivedCurrency, tr.ReceivedAmount, transaction.Type, transaction.Details, link); err != nil {
				return err
			}
		}
	}

	deliveredType := invertOpType(transaction.Type)
	if transaction.DeliveredUserId != nil {
		for _, tr := range transaction.DeliveredOutcomes {
			if err := applyCompanyOp(ctx, cbStorage, cbrStorage, companyID, actingUserID,
				tr.DeliveredCurrency, tr.DeliveredAmount, deliveredType, transaction.Details, link); err != nil {
				return err
			}
		}
	}

	// Yaratish bosqichidagi qarz: payload'da qarzdor kelmasa, eski qarzdagi qarzdor ishlatiladi.
	createDebtInput := debtInput
	if createDebtInput.Phone == "" {
		createDebtInput.Phone = transaction.Phone
	}
	if !createDebtInput.Enabled() {
		createDebtInput = debtInputFromDebts(oldDebts, DEBT_STAGE_CREATE)
	}
	if err := recordTransactionDebt(ctx, tx, cbStorage, cbrStorage, transactionDebtOp{
		CompanyID:     companyID,
		ActingUserID:  actingUserID,
		TransactionID: transaction.ID,
		Stage:         DEBT_STAGE_CREATE,
		Input:         createDebtInput,
		Amounts:       debtAmountsFromIncomes(transaction.ReceivedIncomes),
		DebtType:      int(invertOpType(transaction.Type)),
		BalanceType:   invertOpType(transaction.Type),
		Details:       transaction.Details,
	}); err != nil {
		return fmt.Errorf("debt on transaction update: %w", err)
	}

	// Yakunlash bosqichidagi qarz faqat oldin yozilgan bo'lsa qayta tiklanadi.
	if transaction.DeliveredUserId != nil {
		if err := recordTransactionDebt(ctx, tx, cbStorage, cbrStorage, transactionDebtOp{
			CompanyID:     companyID,
			ActingUserID:  actingUserID,
			TransactionID: transaction.ID,
			Stage:         DEBT_STAGE_COMPLETE,
			Input:         debtInputFromDebts(oldDebts, DEBT_STAGE_COMPLETE),
			Amounts:       debtAmountsFromOutcomes(transaction.DeliveredOutcomes),
			DebtType:      int(invertOpType(deliveredType)),
			BalanceType:   invertOpType(deliveredType),
			Details:       transaction.Details,
		}); err != nil {
			return fmt.Errorf("debt on transaction update (complete stage): %w", err)
		}
	}

	feeCompanyID, err := serviceFeeCompanyForUpdate(ctx, tx, transaction, companyID)
	if err != nil {
		return fmt.Errorf("service fee company on update: %w", err)
	}
	if err := NewServiceFeeService(s.store).SyncFromTransactionTx(
		ctx, tx, transaction, feeCompanyID,
	); err != nil {
		return fmt.Errorf("service fee sync on update: %w", err)
	}

	return tx.Commit()
}

// DeleteTransactionV2 — transaction'ni o'chiradi va company balans ta'sirini bekor qiladi.
// Transaction'dan yaratilgan qarz yozuvlari ham bekor qilinadi (debtor balansi tiklanadi).
func (s *CompanyOpsService) DeleteTransactionV2(ctx context.Context, id int64) error {
	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	transactionsStorage := store.NewTransactionStorage(tx)
	cbStorage := store.NewCompanyBalanceStorage(tx)
	cbrStorage := store.NewCompanyBalanceRecordStorage(tx)

	if err := reverseAndDeleteByLink(ctx, cbStorage, cbrStorage, "transaction_id", id); err != nil {
		return err
	}
	if _, err := reverseTransactionDebts(ctx, tx, cbStorage, cbrStorage, id); err != nil {
		return fmt.Errorf("debt reverse on transaction delete: %w", err)
	}
	if err := transactionsStorage.Delete(ctx, &id); err != nil {
		return fmt.Errorf("ERROR OCCURRED WHILE DELETING TRANSACTION %v", err)
	}

	return tx.Commit()
}

// CreateDebtV2 — debtor + debt yaratadi; received_incomes kompaniya balansiga ta'sir qiladi.
// SELL => chiqim, BUY => kirim. Debtor balansi (debtors jadvali) v1'dagidek yuritiladi.
func (s *CompanyOpsService) CreateDebtV2(ctx context.Context, debt *store.Debts) error {
	if len(debt.ReceivedIncomes) == 0 {
		return fmt.Errorf("received incomes cannot be empty")
	}

	companyID, err := s.companyOf(ctx, debt.UserID)
	if err != nil {
		return err
	}

	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	debtorsStorage := store.NewDebtorsStorage(tx)
	debtsStorage := store.NewDebtsStorage(tx)
	cbStorage := store.NewCompanyBalanceStorage(tx)
	cbrStorage := store.NewCompanyBalanceRecordStorage(tx)

	debtor := &store.Debtors{
		FullName:  debt.FullName,
		Balance:   0,
		Currency:  debt.DebtedCurrency,
		UserID:    debt.UserID,
		CompanyID: companyID,
		Phone:     debt.Phone,
	}
	if err := debtorsStorage.Create(ctx, debtor); err != nil {
		return fmt.Errorf("failed to create debtor: %w", err)
	}

	originalPositiveDebted := debt.DebtedAmount
	switch debt.Type {
	case types.TYPE_SELL:
		debt.DebtedAmount = -debt.DebtedAmount
		debt.State = 1
	case types.TYPE_BUY:
		// stays positive
	default:
		return fmt.Errorf("invalid debt type: %d", debt.Type)
	}
	debt.DebtorID = debtor.ID
	debt.CompanyID = companyID

	if err := debtsStorage.Create(ctx, debt); err != nil {
		return fmt.Errorf("failed to create debt: %w", err)
	}

	link := opLink{DebtId: &debt.ID}
	for _, tr := range debt.ReceivedIncomes {
		if err := applyCompanyOp(ctx, cbStorage, cbrStorage, companyID, debt.UserID,
			tr.ReceivedCurrency, tr.ReceivedAmount, int64(debt.Type), debt.Details, link); err != nil {
			return err
		}
	}

	if debt.Type == types.TYPE_SELL {
		debtor.Balance -= originalPositiveDebted
	} else {
		debtor.Balance += originalPositiveDebted
	}
	if err := debtorsStorage.Update(ctx, debtor); err != nil {
		return fmt.Errorf("failed to update debtor: %w", err)
	}

	return tx.Commit()
}

func absInt64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

// UpdateDebtV2 — debt'ni yangilaydi: eski company balans + debtor ta'sirini bekor qiladi,
// debt qatorini yangilaydi va yangi ta'sirni qayta qo'llaydi. debt.UserID = amalni bajargan hodim.
func (s *CompanyOpsService) UpdateDebtV2(ctx context.Context, debt *store.Debts) error {
	if len(debt.ReceivedIncomes) == 0 {
		return fmt.Errorf("received incomes cannot be empty")
	}

	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	debtsStorage := store.NewDebtsStorage(tx)
	debtorsStorage := store.NewDebtorsStorage(tx)
	cbStorage := store.NewCompanyBalanceStorage(tx)
	cbrStorage := store.NewCompanyBalanceRecordStorage(tx)

	oldDebt, err := debtsStorage.GetByID(ctx, debt.ID)
	if err != nil {
		return fmt.Errorf("failed to get old debt: %w", err)
	}

	debtor, err := debtorsStorage.GetById(ctx, oldDebt.DebtorID)
	if err != nil {
		return fmt.Errorf("failed to get debtor: %w", err)
	}

	companyID := debtor.CompanyID

	// Eski company balans ta'sirini bekor qilamiz
	if err := reverseAndDeleteByLink(ctx, cbStorage, cbrStorage, "debt_id", oldDebt.ID); err != nil {
		return err
	}

	// Eski debtor ta'sirini bekor qilamiz
	oldPositive := absInt64(oldDebt.DebtedAmount)
	if oldDebt.Type == types.TYPE_SELL {
		debtor.Balance += oldPositive
	} else {
		debtor.Balance -= oldPositive
	}

	// Yangi qiymatlarni qo'llaymiz
	originalPositiveDebted := debt.DebtedAmount
	switch debt.Type {
	case types.TYPE_SELL:
		debt.DebtedAmount = -debt.DebtedAmount
	case types.TYPE_BUY:
		// stays positive
	default:
		return fmt.Errorf("invalid debt type: %d", debt.Type)
	}
	debt.DebtorID = debtor.ID
	debt.CompanyID = companyID

	if err := debtsStorage.Update(ctx, debt); err != nil {
		return fmt.Errorf("failed to update debt: %w", err)
	}

	link := opLink{DebtId: &debt.ID}
	for _, tr := range debt.ReceivedIncomes {
		if err := applyCompanyOp(ctx, cbStorage, cbrStorage, companyID, debt.UserID,
			tr.ReceivedCurrency, tr.ReceivedAmount, int64(debt.Type), debt.Details, link); err != nil {
			return err
		}
	}

	if debt.Type == types.TYPE_SELL {
		debtor.Balance -= originalPositiveDebted
	} else {
		debtor.Balance += originalPositiveDebted
	}
	debtor.Currency = debt.DebtedCurrency
	if err := debtorsStorage.Update(ctx, debtor); err != nil {
		return fmt.Errorf("failed to update debtor: %w", err)
	}

	return tx.Commit()
}

// DeleteDebtV2 — debt'ni o'chiradi va company balans + debtor ta'sirini bekor qiladi.
func (s *CompanyOpsService) DeleteDebtV2(ctx context.Context, debtId int64) error {
	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	debtsStorage := store.NewDebtsStorage(tx)
	debtorsStorage := store.NewDebtorsStorage(tx)
	cbStorage := store.NewCompanyBalanceStorage(tx)
	cbrStorage := store.NewCompanyBalanceRecordStorage(tx)

	debt, err := debtsStorage.GetByID(ctx, debtId)
	if err != nil {
		return fmt.Errorf("failed to get debt: %w", err)
	}

	if err := reverseAndDeleteByLink(ctx, cbStorage, cbrStorage, "debt_id", debtId); err != nil {
		return err
	}

	debtor, err := debtorsStorage.GetById(ctx, debt.DebtorID)
	if err != nil {
		return fmt.Errorf("failed to get debtor: %w", err)
	}

	originalPositive := absInt64(debt.DebtedAmount)
	if debt.Type == types.TYPE_SELL {
		debtor.Balance += originalPositive
	} else {
		debtor.Balance -= originalPositive
	}
	if err := debtorsStorage.Update(ctx, debtor); err != nil {
		return fmt.Errorf("failed to update debtor: %w", err)
	}

	if err := debtsStorage.Delete(ctx, debtId); err != nil {
		return fmt.Errorf("failed to delete debt: %w", err)
	}

	return tx.Commit()
}

// DebtTransactionV2 — mavjud debtorga qarz tranzaksiyasi; kompaniya balansiga ta'sir qiladi.
func (s *CompanyOpsService) DebtTransactionV2(ctx context.Context, debt *store.Debts) error {
	if len(debt.ReceivedIncomes) == 0 {
		return fmt.Errorf("received incomes cannot be empty")
	}

	tx, err := s.store.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	debtsStorage := store.NewDebtsStorage(tx)
	debtorsStorage := store.NewDebtorsStorage(tx)
	cbStorage := store.NewCompanyBalanceStorage(tx)
	cbrStorage := store.NewCompanyBalanceRecordStorage(tx)

	debtor, err := debtorsStorage.GetById(ctx, debt.DebtorID)
	if err != nil {
		return fmt.Errorf("failed to get debtor: %w", err)
	}
	if debtor.Currency != debt.DebtedCurrency {
		return fmt.Errorf("debted currencies do not match: %s != %s", debtor.Currency, debt.DebtedCurrency)
	}

	companyID := debtor.CompanyID

	originalPositiveDebted := debt.DebtedAmount
	switch debt.Type {
	case types.TYPE_SELL:
		debt.DebtedAmount = -debt.DebtedAmount
	case types.TYPE_BUY:
		// stays positive
	default:
		return fmt.Errorf("invalid debt type: %d", debt.Type)
	}
	debt.CompanyID = companyID

	if err := debtsStorage.Create(ctx, debt); err != nil {
		return fmt.Errorf("failed to create debt: %w", err)
	}

	link := opLink{DebtId: &debt.ID}
	for _, tr := range debt.ReceivedIncomes {
		if err := applyCompanyOp(ctx, cbStorage, cbrStorage, companyID, debt.UserID,
			tr.ReceivedCurrency, tr.ReceivedAmount, int64(debt.Type), debt.Details, link); err != nil {
			return err
		}
	}

	if debt.Type == types.TYPE_SELL {
		debtor.Balance -= originalPositiveDebted
	} else {
		debtor.Balance += originalPositiveDebted
	}
	if err := debtorsStorage.Update(ctx, debtor); err != nil {
		return fmt.Errorf("failed to update debtor: %w", err)
	}

	return tx.Commit()
}
