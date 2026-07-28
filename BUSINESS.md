# Business (multi-tenant) arxitekturasi

Ierarxiya: **business → company → users**. Har bir business boshqasidan to'liq
izolyatsiya qilingan: bir businessdagi foydalanuvchi hech qanday yo'l bilan
boshqa businessning ma'lumotini ko'ra olmaydi.

## Ma'lumotlar modeli

| Jadval | Yangi ustun | Izoh |
|---|---|---|
| `businesses` | — | Tenant: `name`, `details`, `phone`, `status` |
| `business_settings` | — | Tenant sozlamalari: `transaction_flow` |
| `companies` | `business_id NOT NULL` | Kompaniya = business ichidagi jamoa/filial |
| `users` | `business_id NOT NULL` | User doim bitta business ichida |
| `balances`, `balance_records`, `exchanges`, `transactions`, `debtors`, `debts`, `company_balances`, `company_balance_records`, `soft_balances`, `soft_balance_records`, `transaction_service_fees`, `service_fee_settlements` | `business_id` | Kompaniyadan avtomatik to'ldiriladi (trigger) |

Migratsiyalar:

- `000033_businesses` — `businesses` jadvali, `companies.business_id`,
  `users.business_id`, backfill (mavjud ma'lumot `Default Business`ga o'tadi),
  `UNIQUE (business_id, phone)`, user↔company business mosligini tekshiruvchi trigger.
- `000034_business_scope` — operatsion jadvallarga `business_id`, backfill,
  `INSERT` paytida avtomatik to'ldiruvchi triggerlar, indekslar va
  "tranzaksiya faqat bitta business ichida" cheklovi.
- `000036_business_settings` — `business_settings` jadvali (`transaction_flow`),
  mavjud businesslar oddiy oqimda qoladi; `transactions` ga qabul qilish izlari
  (`accepted_user_id`, `accepted_company_id`, `accepted_at`).

Ishga tushirish:

```
make migrate-up
```

## Izolyatsiya qanday ta'minlanadi

1. **Autentifikatsiya** — `JWTUserMiddleware` DB'dagi userdan `business_id`,
   `company_id`, `role` ni o'qib kontekstga joylaydi. Token ichidagi claim'lar
   faqat mijoz uchun ma'lumot, avtorizatsiyada ishlatilmaydi.
2. **Guard'lar** (`cmd/api/tenant.go`):
   - `authorizeCompany` — so'ralgan kompaniya shu businessdami;
   - `authorizeUser` — so'ralgan user shu businessdami;
   - `authorizeResource` — `{id}` bo'yicha kelgan yozuv (`transactions`,
     `exchanges`, `debts`, `debtors`, `balance_records`, ...) shu businessdami;
   - `authorizeFilter` — mijoz yuborgan `field_name`/`field_value` juftligi;
   - `authorizeTransactionCompanies` — tranzaksiyaning ikkala tomoni ham shu business ichida.
3. **So'rov filtri** — barcha ro'yxat so'rovlari `WHERE business_id = $1` bilan
   ketadi (`Archived`, `GetByField`, `ListAll`, `GetAll`, ...).
4. **DB darajasi** — triggerlar `business_id` ni kompaniyadan to'ldiradi va
   businesslar orasidagi tranzaksiyani `RAISE EXCEPTION` bilan bloklaydi.

`field_name` endi oq ro'yxat orqali tekshiriladi va parametr sifatida uzatiladi
(avval so'rov matniga to'g'ridan-to'g'ri qo'shilardi).

## Rollar

| Rol | Qiymat | Huquq |
|---|---|---|
| `ROLE_OWNER` | 1 | Business egasi: business ichidagi barcha kompaniyalar |
| `ROLE_STAFF` | 2 | Hodim: faqat o'z kompaniyasi |

Faqat ega qila oladi: hodim yaratish/o'chirish, rol berish, boshqa userga push
yuborish va **o'z businessi ichida kompaniya (jamoa) ochish/yangilash/o'chirish**.

## API o'zgarishlari

Business **faqat platforma REST API'si orqali** ochiladi (`X-Admin-Key`).
Kompaniya ikki joyda boshqariladi: platforma admin API'si (istalgan business
uchun) va business egasining tokeni (faqat o'z businessi ichida).

### Platforma admin API (`X-Admin-Key: $ADMIN_API_KEY`)

| Method | Path | Izoh |
|---|---|---|
| POST | `/api/v1/admin/businesses` | Business + birinchi kompaniya + egasi (bitta tranzaksiya) |
| GET | `/api/v1/admin/businesses/{id}` | Business ma'lumoti |
| PUT | `/api/v1/admin/businesses/{id}` | Business ma'lumotini yangilash |
| POST | `/api/v1/admin/companies` | Mavjud business ichida yangi kompaniya |
| PUT | `/api/v1/admin/companies/{id}` | Kompaniyani yangilash |
| DELETE | `/api/v1/admin/companies/{id}` | Kompaniyani o'chirish |

`ADMIN_API_KEY` env'i sozlanmagan bo'lsa bu yo'llar `503 ADMIN API DISABLED`
qaytaradi; kalit noto'g'ri bo'lsa `401 INVALID ADMIN KEY`.

```bash
# Yangi business
curl -X POST https://api.example.com/api/v1/admin/businesses \
  -H "X-Admin-Key: $ADMIN_API_KEY" -H 'Content-Type: application/json' \
  -d '{
        "name": "Hisobchi LLC",
        "details": "Toshkent",
        "phone": "90 123 45 67",
        "company_name": "Chilonzor filiali",
        "owner": {"username": "Ali", "phone": "90 123 45 67", "password": "..."}
      }'

# Shu business ichida yana bir kompaniya (jamoa)
curl -X POST https://api.example.com/api/v1/admin/companies \
  -H "X-Admin-Key: $ADMIN_API_KEY" -H 'Content-Type: application/json' \
  -d '{"business_id": 3, "name": "Yunusobod filiali"}'
```

## Tranzaksiya oqimi (business sozlamasi)

`business_settings.transaction_flow` har bir business uchun oqimni belgilaydi:

| Qiymat | Oqim | Bosqichlar |
|---|---|---|
| `1` (standart) | `SIMPLE` | yaratish → topshirish |
| `2` | `THREE_STAGE` | yaratish → **qabul qilish** → topshirish |

**Qabul qilish bosqichi balansga ta'sir qilmaydi.** Pul harakati ikkala oqimda
ham bir xil joyda hisoblanadi:

- yaratish (`POST /transactions/create/v2`) — `received_incomes` kompaniya balansiga;
- topshirish (`POST /transactions/complete/v2`) — `delivered_outcomes` kompaniya balansiga.

Qabul qilish faqat `transactions.status` ni `1 → 4` ga o'tkazadi va kim/qachon
qabul qilganini yozadi. Status qiymatlari: `1` yaratilgan, `2` yakunlangan,
`3` arxivlangan, `4` qabul qilingan.

`THREE_STAGE` yoqilgan bo'lsa yakunlash (`complete` va `complete/v2`) faqat
qabul qilingan tranzaksiya uchun ishlaydi, aks holda `400 BUYURTMA HALI QABUL
QILINMAGAN`. `SIMPLE` oqimda `accept/v2` chaqirilsa `400 QABUL QILISH BOSQICHI
SOZLAMALARDA YOQILMAGAN`.

| Method | Path | Kim |
|---|---|---|
| GET | `/api/v1/user/business/settings` | Token — o'qish (mobil oqimni bilishi uchun) |
| PUT | `/api/v1/user/business/settings` | Faqat business egasi |
| POST | `/api/v1/user/transactions/accept/v2` | `{"transactionID": 12}` — qabul qilish |

Ro'yxat javoblarida qo'shimcha maydonlar: `accepted_user_id`, `accepted_user`,
`accepted_company_id`, `accepted_at`, `is_accepted`.

## Xizmat haqi ko'rinishi

Xizmat haqi tranzaksiyaning **ikkala tomoniga ham** ko'rinadi: Toshkent olgan
xizmat haqi Namangan ro'yxatida ham, Namangan olgani Toshkentda ham chiqadi
(avval hodimga faqat o'z kompaniyasi olgan haq ko'rinardi, qolgani `0` bo'lib
maskalanardi).

Tranzaksiya ro'yxatlaridagi maydonlar (`GetByField`, `Archived`,
`GetByCompanyId`): `service_fee_amount`, `service_fee_currency`,
`service_fee_details` va kim olgani — `service_fee_company_id`,
`service_fee_company`. Kompaniya `transaction_service_fees.company_id` dan
olinadi; eski yozuvda u bo'lmasa, yaratishda kiritilgan bo'lsa qabul qiluvchi,
yakunlashda kiritilgan bo'lsa yetkazuvchi kompaniya hisoblanadi.

Info karta (`GetInfos`) avvalgidek qoladi: hodim faqat o'z kompaniyasining
xizmat haqi summasini ko'radi.

```bash
# 3 bosqichli oqimni yoqish
curl -X PUT https://api.example.com/api/v1/user/business/settings \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"transaction_flow": 2}'
```

### Mijoz ilovasi uchun

| Method | Path | Kim |
|---|---|---|
| POST | `/api/v1/users/login` | Ochiq |
| POST | `/api/v1/users/refresh` | Ochiq |
| GET | `/api/v1/users/businesses` | Token — parolsiz o'tish mumkin bo'lgan businesslar |
| POST | `/api/v1/users/switch-business` | Token — boshqa businessga o'tish |
| POST | `/api/v1/users/register`, `/api/v1/user/` | Business egasi (token) — hodim qo'shish |
| GET | `/api/v1/user/business` | Token — o'z businessi |
| GET | `/api/v1/user/business/team` | Token — jamoalar (hodimga faqat o'z kompaniyasi) |
| GET | `/api/v1/user/companies/all`, `/{id}` | Token — o'qish (hodim ham) |
| POST | `/api/v1/user/companies` | Faqat business egasi — yangi kompaniya |
| PUT | `/api/v1/user/companies/{id}` | Faqat business egasi |
| DELETE | `/api/v1/user/companies/{id}` | Faqat business egasi, bo'sh kompaniya |

Egasining kompaniya CRUD'i:

- `business_id` mijozdan olinmaydi — doim tokendagi businessdan qo'yiladi,
  shuning uchun begona businessga kompaniya qo'shib bo'lmaydi;
- yaratishda kompaniya + `company_balances` + `soft_balances` standart qatorlari
  bitta tranzaksiyada ochiladi (`USD`, `SUM`);
- yangilashda bo'sh yuborilgan `name`/`password` eskisicha qoladi;
- o'chirish faqat **bo'sh** kompaniya uchun: faol hodim, tranzaksiya, ayirboshlash
  yoki qarzdor bo'lsa `400 KOMPANIYADA HODIM YOKI OPERATSIYA BOR ...`, o'zi
  turgan kompaniyani o'chirsa `400 O'ZINGIZ TURGAN KOMPANIYANI O'CHIRIB
  BO'LMAYDI`. Bu cheklov kaskadli `ON DELETE CASCADE` orqali balans va xizmat
  haqi tarixi yo'qolib ketmasligi uchun.

```bash
# Ega o'z businessida yangi jamoa ochadi
curl -X POST https://api.example.com/api/v1/user/companies \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"name": "Yunusobod filiali", "details": "Toshkent", "password": ""}'
```

### Mobil uchun muhim

- `POST /api/v1/users/register` endi **token talab qiladi** va faqat business
  egasi chaqira oladi.
- `POST /api/v1/users/login` — telefon business ichida unikal, businesslar
  bo'ylab takrorlanishi mumkin. Bir odam bir nechta businessda bo'lsa login
  **xato bermaydi**: birinchi business ochiladi, qolganlari javobdagi
  `businesses` ro'yxatida qaytadi. So'rovga ixtiyoriy `business_id` qo'shilsa
  to'g'ridan-to'g'ri o'sha business ochiladi:
  ```json
  { "phone": "90 123 45 67", "password": "...", "business_id": 3 }
  ```
- Login/refresh javobidagi `user` obyektida `business_id` bor; token ichida
  `businessID`, `companyID`, `role` claim'lari qaytadi.
- Boshqa business/kompaniya resursiga murojaat `403` qaytaradi.
- "Admin" (`role=1`) endi **platforma bo'ylab emas, business bo'ylab** hamma
  narsani ko'radi: xizmat haqi ro'yxatlari, info karta, arxiv — barchasi
  business bilan chegaralangan.

## Business almashtirish (parolsiz)

Bir odam bir nechta businessda ishlashi mumkin: har business uchun **alohida
`users` qatori**, lekin telefon bir xil. Login/refresh/switch javoblarida shu
odam kira oladigan businesslar ro'yxati qaytadi:

```json
{
  "token": "...", "refresh_token": "...", "user": { "id": 12, "business_id": 1 },
  "businesses": [
    {"business_id": 1, "business_name": "Hisobchi LLC", "business_status": 1,
     "user_id": 12, "username": "Ali", "company_id": 4, "role": 1},
    {"business_id": 3, "business_name": "Sarrof MChJ", "business_status": 1,
     "user_id": 47, "username": "Ali", "company_id": 9, "role": 2}
  ]
}
```

Almashish parol so'ramaydi — yangi token juftligi beriladi:

```bash
curl -X POST https://api.example.com/api/v1/users/switch-business \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"business_id": 3}'
```

Mijoz eski token/refresh tokenni tashlab, javobdagi yangilariga o'tadi.

**Ruxsat qoidasi:** almashish faqat **bir xil telefon + bir xil parol**ga ega
qatorlar orasida mumkin. Ro'yxat har safar DB'dan jonli hisoblanadi
(`LoginCandidates(phone, password)`), shuning uchun:

- boshqa odamning businessi ro'yxatga tushmaydi (telefon boshqa);
- bir businessdagi parol o'zgartirilsa, o'sha business ro'yxatdan darhol
  chiqadi — eski token bilan ham o'tib bo'lmaydi;
- ruxsatsiz `business_id` uchun `403 BU BUSINESSGA RUXSAT YO'Q`.

Ya'ni parollarni turli businessda **bir xil saqlash** shart: farqli parol =
alohida hisob, almashish yo'q, qayta login qilinadi.

O'chirilgan userlarning telefoni bo'sh (`phone = ''`) bo'lgani uchun ular hech
qachon almashish ro'yxatiga tushmaydi.
