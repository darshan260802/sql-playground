-- Core dataset schema. Executed inside the target schema (core, or a user's copy).
-- Table comments start with "<Domain>:" so the UI can group tables in the schema browser.

-- ---------------------------------------------------------------- E-commerce

CREATE TABLE categories (
  category_id        SMALLINT UNSIGNED NOT NULL COMMENT 'Primary key',
  name               VARCHAR(60) NOT NULL COMMENT 'Category name',
  parent_category_id SMALLINT UNSIGNED NULL COMMENT 'Parent category (NULL for top level)',
  PRIMARY KEY (category_id),
  KEY idx_categories_parent (parent_category_id),
  CONSTRAINT fk_categories_parent FOREIGN KEY (parent_category_id) REFERENCES categories (category_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci
  COMMENT='E-commerce: product category tree (up to 3 levels deep)';

CREATE TABLE suppliers (
  supplier_id   SMALLINT UNSIGNED NOT NULL COMMENT 'Primary key',
  name          VARCHAR(80) NOT NULL COMMENT 'Supplier company name',
  country       VARCHAR(40) NOT NULL COMMENT 'Country of the supplier',
  contact_email VARCHAR(120) NULL COMMENT 'Contact email (may be missing)',
  rating        DECIMAL(2,1) NOT NULL COMMENT 'Internal supplier rating 1.0-5.0',
  since         DATE NOT NULL COMMENT 'Date the partnership started',
  PRIMARY KEY (supplier_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci
  COMMENT='E-commerce: companies that supply products';

CREATE TABLE products (
  product_id   INT UNSIGNED NOT NULL COMMENT 'Primary key',
  sku          VARCHAR(20) NOT NULL COMMENT 'Stock keeping unit, unique',
  name         VARCHAR(120) NOT NULL COMMENT 'Product name',
  category_id  SMALLINT UNSIGNED NOT NULL COMMENT 'Leaf category',
  supplier_id  SMALLINT UNSIGNED NOT NULL COMMENT 'Supplier',
  brand        VARCHAR(40) NOT NULL COMMENT 'Brand name',
  unit_price   DECIMAL(10,2) NOT NULL COMMENT 'Current list price in USD',
  unit_cost    DECIMAL(10,2) NOT NULL COMMENT 'Current unit cost in USD',
  attributes   JSON NULL COMMENT 'Extra attributes, e.g. {"color": "black", "weight_kg": 0.3}',
  launched_on  DATE NOT NULL COMMENT 'Date the product went on sale',
  is_active    TINYINT(1) NOT NULL DEFAULT 1 COMMENT '1 if currently sold, 0 if discontinued',
  PRIMARY KEY (product_id),
  UNIQUE KEY uq_products_sku (sku),
  KEY idx_products_category (category_id),
  KEY idx_products_supplier (supplier_id),
  CONSTRAINT fk_products_category FOREIGN KEY (category_id) REFERENCES categories (category_id),
  CONSTRAINT fk_products_supplier FOREIGN KEY (supplier_id) REFERENCES suppliers (supplier_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci
  COMMENT='E-commerce: product catalog';

CREATE TABLE customers (
  customer_id             INT UNSIGNED NOT NULL COMMENT 'Primary key',
  first_name              VARCHAR(40) NOT NULL COMMENT 'First name',
  last_name               VARCHAR(40) NOT NULL COMMENT 'Last name',
  email                   VARCHAR(120) NOT NULL COMMENT 'Email address, unique',
  phone                   VARCHAR(30) NULL COMMENT 'Phone number (often missing)',
  gender                  CHAR(1) NULL COMMENT 'F, M or NULL when not provided',
  birth_date              DATE NULL COMMENT 'Date of birth (may be missing)',
  city                    VARCHAR(40) NOT NULL COMMENT 'City',
  state                   VARCHAR(40) NULL COMMENT 'State / region',
  country                 VARCHAR(40) NOT NULL COMMENT 'Country',
  signup_at               DATETIME NOT NULL COMMENT 'When the account was created',
  loyalty_tier            ENUM('bronze','silver','gold','platinum') NOT NULL DEFAULT 'bronze' COMMENT 'Loyalty program tier',
  acquisition_channel     VARCHAR(20) NOT NULL COMMENT 'How the customer found us',
  referred_by_customer_id INT UNSIGNED NULL COMMENT 'Customer who referred this one (self reference)',
  PRIMARY KEY (customer_id),
  UNIQUE KEY uq_customers_email (email),
  KEY idx_customers_country_city (country, city),
  KEY idx_customers_referred_by (referred_by_customer_id),
  CONSTRAINT fk_customers_referred_by FOREIGN KEY (referred_by_customer_id) REFERENCES customers (customer_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci
  COMMENT='E-commerce: registered customers (also own bank accounts in Finance)';

CREATE TABLE orders (
  order_id     INT UNSIGNED NOT NULL COMMENT 'Primary key',
  customer_id  INT UNSIGNED NOT NULL COMMENT 'Customer who placed the order',
  order_date   DATETIME NOT NULL COMMENT 'When the order was placed',
  status       ENUM('pending','processing','shipped','delivered','cancelled','returned') NOT NULL COMMENT 'Order status',
  channel      ENUM('web','mobile','store') NOT NULL COMMENT 'Sales channel',
  coupon_code  VARCHAR(20) NULL COMMENT 'Coupon used (NULL if none)',
  shipping_fee DECIMAL(8,2) NOT NULL COMMENT 'Shipping fee charged in USD',
  PRIMARY KEY (order_id),
  KEY idx_orders_customer (customer_id),
  KEY idx_orders_date (order_date),
  CONSTRAINT fk_orders_customer FOREIGN KEY (customer_id) REFERENCES customers (customer_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci
  COMMENT='E-commerce: customer orders (order header)';

CREATE TABLE order_items (
  order_item_id INT UNSIGNED NOT NULL COMMENT 'Primary key',
  order_id      INT UNSIGNED NOT NULL COMMENT 'Order this line belongs to',
  product_id    INT UNSIGNED NOT NULL COMMENT 'Product purchased',
  quantity      SMALLINT UNSIGNED NOT NULL COMMENT 'Units purchased',
  unit_price    DECIMAL(10,2) NOT NULL COMMENT 'Price per unit at time of purchase (USD)',
  discount      DECIMAL(4,2) NOT NULL DEFAULT 0.00 COMMENT 'Discount as a fraction, e.g. 0.15 = 15% off',
  PRIMARY KEY (order_item_id),
  KEY idx_order_items_order (order_id),
  KEY idx_order_items_product (product_id),
  CONSTRAINT fk_order_items_order FOREIGN KEY (order_id) REFERENCES orders (order_id),
  CONSTRAINT fk_order_items_product FOREIGN KEY (product_id) REFERENCES products (product_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci
  COMMENT='E-commerce: order lines. Line revenue = quantity * unit_price * (1 - discount)';

CREATE TABLE payments (
  payment_id     INT UNSIGNED NOT NULL COMMENT 'Primary key',
  order_id       INT UNSIGNED NOT NULL COMMENT 'Order being paid',
  payment_method ENUM('card','paypal','bank_transfer','gift_card','cash_on_delivery') NOT NULL COMMENT 'Payment method',
  amount         DECIMAL(10,2) NOT NULL COMMENT 'Amount in USD',
  paid_at        DATETIME NOT NULL COMMENT 'When the payment was attempted',
  status         ENUM('paid','failed','refunded') NOT NULL COMMENT 'paid, failed, or refunded (money returned)',
  PRIMARY KEY (payment_id),
  KEY idx_payments_order (order_id),
  CONSTRAINT fk_payments_order FOREIGN KEY (order_id) REFERENCES orders (order_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci
  COMMENT='E-commerce: payment attempts and refunds (an order can have several)';

CREATE TABLE shipments (
  shipment_id  INT UNSIGNED NOT NULL COMMENT 'Primary key',
  order_id     INT UNSIGNED NOT NULL COMMENT 'Order shipped',
  carrier      VARCHAR(20) NOT NULL COMMENT 'Shipping carrier',
  shipped_at   DATETIME NOT NULL COMMENT 'When the parcel left the warehouse',
  delivered_at DATETIME NULL COMMENT 'When it was delivered (NULL if still in transit)',
  PRIMARY KEY (shipment_id),
  UNIQUE KEY uq_shipments_order (order_id),
  CONSTRAINT fk_shipments_order FOREIGN KEY (order_id) REFERENCES orders (order_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci
  COMMENT='E-commerce: shipments (store pickups, cancelled and most pending orders have none)';

CREATE TABLE reviews (
  review_id     INT UNSIGNED NOT NULL COMMENT 'Primary key',
  product_id    INT UNSIGNED NOT NULL COMMENT 'Product reviewed',
  customer_id   INT UNSIGNED NOT NULL COMMENT 'Customer who wrote the review',
  rating        TINYINT UNSIGNED NOT NULL COMMENT 'Star rating 1-5',
  review_date   DATE NOT NULL COMMENT 'Date the review was posted',
  helpful_votes SMALLINT UNSIGNED NOT NULL DEFAULT 0 COMMENT 'How many people found it helpful',
  PRIMARY KEY (review_id),
  KEY idx_reviews_product (product_id),
  KEY idx_reviews_customer (customer_id),
  CONSTRAINT fk_reviews_product FOREIGN KEY (product_id) REFERENCES products (product_id),
  CONSTRAINT fk_reviews_customer FOREIGN KEY (customer_id) REFERENCES customers (customer_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci
  COMMENT='E-commerce: product reviews';

-- ---------------------------------------------------------------- HR

CREATE TABLE offices (
  office_id SMALLINT UNSIGNED NOT NULL COMMENT 'Primary key',
  city      VARCHAR(40) NOT NULL COMMENT 'City',
  country   VARCHAR(40) NOT NULL COMMENT 'Country',
  region    VARCHAR(20) NOT NULL COMMENT 'Region: AMER, EMEA or APAC',
  opened_on DATE NOT NULL COMMENT 'Date the office opened',
  PRIMARY KEY (office_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci
  COMMENT='HR: company offices';

CREATE TABLE departments (
  department_id SMALLINT UNSIGNED NOT NULL COMMENT 'Primary key',
  name          VARCHAR(40) NOT NULL COMMENT 'Department name',
  budget        DECIMAL(12,2) NOT NULL COMMENT 'Annual budget in USD',
  PRIMARY KEY (department_id),
  UNIQUE KEY uq_departments_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci
  COMMENT='HR: departments';

CREATE TABLE jobs (
  job_id     SMALLINT UNSIGNED NOT NULL COMMENT 'Primary key',
  title      VARCHAR(60) NOT NULL COMMENT 'Job title',
  job_family VARCHAR(30) NOT NULL COMMENT 'Job family, e.g. Engineering',
  level      TINYINT UNSIGNED NOT NULL COMMENT 'Seniority level 1 (junior) - 8 (executive)',
  min_salary DECIMAL(10,2) NOT NULL COMMENT 'Salary band minimum (USD)',
  max_salary DECIMAL(10,2) NOT NULL COMMENT 'Salary band maximum (USD)',
  PRIMARY KEY (job_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci
  COMMENT='HR: job titles with salary bands';

CREATE TABLE employees (
  employee_id      INT UNSIGNED NOT NULL COMMENT 'Primary key',
  first_name       VARCHAR(40) NOT NULL COMMENT 'First name',
  last_name        VARCHAR(40) NOT NULL COMMENT 'Last name',
  email            VARCHAR(120) NOT NULL COMMENT 'Work email, unique',
  gender           CHAR(1) NOT NULL COMMENT 'F or M',
  birth_date       DATE NOT NULL COMMENT 'Date of birth',
  hire_date        DATE NOT NULL COMMENT 'Date hired',
  termination_date DATE NULL COMMENT 'Date the employee left (NULL if still employed)',
  department_id    SMALLINT UNSIGNED NOT NULL COMMENT 'Department',
  job_id           SMALLINT UNSIGNED NOT NULL COMMENT 'Current job',
  manager_id       INT UNSIGNED NULL COMMENT 'Direct manager (NULL for the CEO)',
  office_id        SMALLINT UNSIGNED NOT NULL COMMENT 'Office location',
  employment_type  ENUM('full_time','part_time','contractor') NOT NULL COMMENT 'Employment type',
  PRIMARY KEY (employee_id),
  UNIQUE KEY uq_employees_email (email),
  KEY idx_employees_department (department_id),
  KEY idx_employees_job (job_id),
  KEY idx_employees_manager (manager_id),
  KEY idx_employees_office (office_id),
  CONSTRAINT fk_employees_department FOREIGN KEY (department_id) REFERENCES departments (department_id),
  CONSTRAINT fk_employees_job FOREIGN KEY (job_id) REFERENCES jobs (job_id),
  CONSTRAINT fk_employees_manager FOREIGN KEY (manager_id) REFERENCES employees (employee_id),
  CONSTRAINT fk_employees_office FOREIGN KEY (office_id) REFERENCES offices (office_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci
  COMMENT='HR: employees with a manager hierarchy (self reference)';

CREATE TABLE salaries (
  employee_id INT UNSIGNED NOT NULL COMMENT 'Employee',
  salary      DECIMAL(10,2) NOT NULL COMMENT 'Annual salary in USD',
  from_date   DATE NOT NULL COMMENT 'Salary effective from (inclusive)',
  to_date     DATE NULL COMMENT 'Salary effective until (exclusive); NULL = current salary',
  PRIMARY KEY (employee_id, from_date),
  CONSTRAINT fk_salaries_employee FOREIGN KEY (employee_id) REFERENCES employees (employee_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci
  COMMENT='HR: salary history, one row per salary period';

CREATE TABLE performance_reviews (
  review_id   INT UNSIGNED NOT NULL COMMENT 'Primary key',
  employee_id INT UNSIGNED NOT NULL COMMENT 'Employee reviewed',
  review_year SMALLINT UNSIGNED NOT NULL COMMENT 'Year the review covers',
  rating      TINYINT UNSIGNED NOT NULL COMMENT 'Rating 1 (poor) - 5 (outstanding)',
  reviewer_id INT UNSIGNED NULL COMMENT 'Employee who wrote the review (usually the manager)',
  PRIMARY KEY (review_id),
  UNIQUE KEY uq_perf_employee_year (employee_id, review_year),
  KEY idx_perf_reviewer (reviewer_id),
  CONSTRAINT fk_perf_employee FOREIGN KEY (employee_id) REFERENCES employees (employee_id),
  CONSTRAINT fk_perf_reviewer FOREIGN KEY (reviewer_id) REFERENCES employees (employee_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci
  COMMENT='HR: yearly performance reviews';

CREATE TABLE leave_requests (
  leave_id    INT UNSIGNED NOT NULL COMMENT 'Primary key',
  employee_id INT UNSIGNED NOT NULL COMMENT 'Employee',
  leave_type  ENUM('vacation','sick','parental','unpaid','bereavement') NOT NULL COMMENT 'Type of leave',
  start_date  DATE NOT NULL COMMENT 'First day of leave',
  end_date    DATE NOT NULL COMMENT 'Last day of leave (inclusive)',
  status      ENUM('approved','rejected','pending') NOT NULL COMMENT 'Request status',
  PRIMARY KEY (leave_id),
  KEY idx_leave_employee (employee_id),
  CONSTRAINT fk_leave_employee FOREIGN KEY (employee_id) REFERENCES employees (employee_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci
  COMMENT='HR: leave / time-off requests';

-- ---------------------------------------------------------------- Finance

CREATE TABLE stocks (
  stock_id           SMALLINT UNSIGNED NOT NULL COMMENT 'Primary key',
  ticker             VARCHAR(8) NOT NULL COMMENT 'Ticker symbol (fictional companies)',
  company_name       VARCHAR(80) NOT NULL COMMENT 'Company name',
  sector             VARCHAR(40) NOT NULL COMMENT 'Sector',
  industry           VARCHAR(60) NOT NULL COMMENT 'Industry',
  exchange           VARCHAR(10) NOT NULL COMMENT 'Listing exchange',
  ipo_date           DATE NOT NULL COMMENT 'First trading day',
  delisted_on        DATE NULL COMMENT 'Last trading day if delisted',
  shares_outstanding BIGINT UNSIGNED NOT NULL COMMENT 'Shares outstanding',
  PRIMARY KEY (stock_id),
  UNIQUE KEY uq_stocks_ticker (ticker)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci
  COMMENT='Finance: listed companies (fictional)';

CREATE TABLE stock_prices (
  stock_id   SMALLINT UNSIGNED NOT NULL COMMENT 'Stock',
  trade_date DATE NOT NULL COMMENT 'Trading day (weekdays, excluding market holidays)',
  open       DECIMAL(12,4) NOT NULL COMMENT 'Opening price (USD)',
  high       DECIMAL(12,4) NOT NULL COMMENT 'Daily high',
  low        DECIMAL(12,4) NOT NULL COMMENT 'Daily low',
  close      DECIMAL(12,4) NOT NULL COMMENT 'Closing price',
  volume     INT UNSIGNED NOT NULL COMMENT 'Shares traded',
  PRIMARY KEY (stock_id, trade_date),
  CONSTRAINT fk_stock_prices_stock FOREIGN KEY (stock_id) REFERENCES stocks (stock_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci
  COMMENT='Finance: daily OHLCV prices';

CREATE TABLE accounts (
  account_id   INT UNSIGNED NOT NULL COMMENT 'Primary key',
  customer_id  INT UNSIGNED NOT NULL COMMENT 'Account holder (links to E-commerce customers)',
  account_type ENUM('checking','savings','credit','brokerage') NOT NULL COMMENT 'Account type',
  currency     CHAR(3) NOT NULL COMMENT 'ISO currency code',
  opened_on    DATE NOT NULL COMMENT 'Date opened',
  closed_on    DATE NULL COMMENT 'Date closed (NULL if open)',
  credit_limit DECIMAL(10,2) NULL COMMENT 'Credit limit in account currency (credit accounts only)',
  PRIMARY KEY (account_id),
  KEY idx_accounts_customer (customer_id),
  CONSTRAINT fk_accounts_customer FOREIGN KEY (customer_id) REFERENCES customers (customer_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci
  COMMENT='Finance: bank and brokerage accounts';

CREATE TABLE transactions (
  txn_id            INT UNSIGNED NOT NULL COMMENT 'Primary key',
  account_id        INT UNSIGNED NOT NULL COMMENT 'Account',
  txn_at            DATETIME NOT NULL COMMENT 'Timestamp',
  amount            DECIMAL(12,2) NOT NULL COMMENT 'Signed amount in account currency (negative = money out)',
  txn_type          ENUM('deposit','salary','withdrawal','purchase','transfer_in','transfer_out','fee','interest','refund') NOT NULL COMMENT 'Transaction type',
  merchant          VARCHAR(60) NULL COMMENT 'Merchant (purchases and refunds only)',
  merchant_category VARCHAR(30) NULL COMMENT 'Merchant category',
  PRIMARY KEY (txn_id),
  KEY idx_transactions_account_time (account_id, txn_at),
  CONSTRAINT fk_transactions_account FOREIGN KEY (account_id) REFERENCES accounts (account_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci
  COMMENT='Finance: account transactions. Balance = running SUM(amount)';

CREATE TABLE trades (
  trade_id    INT UNSIGNED NOT NULL COMMENT 'Primary key',
  account_id  INT UNSIGNED NOT NULL COMMENT 'Brokerage account',
  stock_id    SMALLINT UNSIGNED NOT NULL COMMENT 'Stock traded',
  side        ENUM('buy','sell') NOT NULL COMMENT 'Buy or sell',
  quantity    INT UNSIGNED NOT NULL COMMENT 'Number of shares',
  price       DECIMAL(12,4) NOT NULL COMMENT 'Execution price per share (USD)',
  executed_at DATETIME NOT NULL COMMENT 'Execution timestamp',
  PRIMARY KEY (trade_id),
  KEY idx_trades_account (account_id, executed_at),
  KEY idx_trades_stock (stock_id),
  CONSTRAINT fk_trades_account FOREIGN KEY (account_id) REFERENCES accounts (account_id),
  CONSTRAINT fk_trades_stock FOREIGN KEY (stock_id) REFERENCES stocks (stock_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci
  COMMENT='Finance: stock trades executed in brokerage accounts';

CREATE TABLE exchange_rates (
  currency    CHAR(3) NOT NULL COMMENT 'ISO currency code',
  rate_date   DATE NOT NULL COMMENT 'Business day of the rate',
  rate_to_usd DECIMAL(14,8) NOT NULL COMMENT 'USD value of 1 unit of the currency',
  PRIMARY KEY (currency, rate_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci
  COMMENT='Finance: daily FX rates to USD (business days only, no weekends)';

CREATE TABLE loans (
  loan_id       INT UNSIGNED NOT NULL COMMENT 'Primary key',
  customer_id   INT UNSIGNED NOT NULL COMMENT 'Borrower',
  loan_type     ENUM('personal','auto','mortgage','student') NOT NULL COMMENT 'Loan type',
  principal     DECIMAL(12,2) NOT NULL COMMENT 'Amount borrowed (USD)',
  annual_rate   DECIMAL(5,4) NOT NULL COMMENT 'Annual interest rate, e.g. 0.0650 = 6.5%',
  term_months   SMALLINT UNSIGNED NOT NULL COMMENT 'Term in months',
  start_date    DATE NOT NULL COMMENT 'Disbursement date',
  status        ENUM('active','paid_off','defaulted') NOT NULL COMMENT 'Loan status',
  PRIMARY KEY (loan_id),
  KEY idx_loans_customer (customer_id),
  CONSTRAINT fk_loans_customer FOREIGN KEY (customer_id) REFERENCES customers (customer_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci
  COMMENT='Finance: customer loans';

-- ---------------------------------------------------------------- Utility

CREATE TABLE calendar (
  date_key      DATE NOT NULL COMMENT 'The date',
  year          SMALLINT UNSIGNED NOT NULL COMMENT 'Year',
  quarter       TINYINT UNSIGNED NOT NULL COMMENT 'Quarter 1-4',
  month         TINYINT UNSIGNED NOT NULL COMMENT 'Month 1-12',
  month_name    VARCHAR(9) NOT NULL COMMENT 'Month name',
  day_of_month  TINYINT UNSIGNED NOT NULL COMMENT 'Day of month',
  day_of_week   TINYINT UNSIGNED NOT NULL COMMENT 'ISO day of week, 1 = Monday',
  day_name      VARCHAR(9) NOT NULL COMMENT 'Day name',
  iso_week      TINYINT UNSIGNED NOT NULL COMMENT 'ISO week number',
  is_weekend    TINYINT(1) NOT NULL COMMENT '1 on Saturday and Sunday',
  is_holiday    TINYINT(1) NOT NULL COMMENT '1 on US market holidays',
  PRIMARY KEY (date_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci
  COMMENT='Utility: one row per day 2012-2030, handy for filling date gaps';
