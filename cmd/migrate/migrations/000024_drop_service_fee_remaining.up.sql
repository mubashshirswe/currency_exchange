-- remaining_amount olib tashlanadi — faqat amount ishlatiladi.
ALTER TABLE transaction_service_fees DROP COLUMN IF EXISTS remaining_amount;
