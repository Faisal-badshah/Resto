-- Base schema and sample data (initial)
DROP TABLE IF EXISTS admins;
DROP TABLE IF EXISTS subscribers;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS reviews;
DROP TABLE IF EXISTS galleries;
DROP TABLE IF EXISTS menus;
DROP TABLE IF EXISTS restaurants;

CREATE TABLE restaurants (
  id SERIAL PRIMARY KEY,
  name TEXT,
  story TEXT,
  address TEXT,
  phone TEXT,
  email TEXT,
  hours TEXT,
  social_links JSONB DEFAULT '[]',
  offerings JSONB DEFAULT '[]',
  site_config JSONB DEFAULT '{}'
);

CREATE TABLE menus (
  id SERIAL PRIMARY KEY,
  restaurant_id INT REFERENCES restaurants(id),
  category TEXT,
  items_json JSONB,
  UNIQUE(restaurant_id, category)
);

CREATE TABLE galleries (
  id SERIAL PRIMARY KEY,
  restaurant_id INT REFERENCES restaurants(id) UNIQUE,
  images JSONB DEFAULT '[]',
  captions JSONB DEFAULT '[]'
);

CREATE TABLE reviews (
  id SERIAL PRIMARY KEY,
  restaurant_id INT REFERENCES restaurants(id) UNIQUE,
  testimonials JSONB DEFAULT '[]'
);

CREATE TABLE orders (
  id SERIAL PRIMARY KEY,
  restaurant_id INT REFERENCES restaurants(id),
  items_json JSONB,
  total NUMERIC(10,2),
  status TEXT,
  created_at TIMESTAMP WITH TIME ZONE,
  customer_name TEXT,
  customer_phone TEXT,
  customer_address TEXT,
  customer_email TEXT,
  notes TEXT
);

CREATE TABLE subscribers (
  id SERIAL PRIMARY KEY,
  restaurant_id INT REFERENCES restaurants(id),
  email TEXT UNIQUE
);

CREATE TABLE admins (
  id SERIAL PRIMARY KEY,
  restaurant_id INT REFERENCES restaurants(id),
  email TEXT,
  password_hash TEXT,
  role TEXT,
  permissions JSONB
);

-- Seed sample restaurant
INSERT INTO restaurants (id, name, story, address, phone, email, hours, social_links, offerings, site_config)
VALUES (
  1,
  'Sample Restaurant',
  'We are a family-run kitchen serving seasonal dishes with a local twist.',
  '123 Food St, Cityville',
  '+15551234567',
  'owner@sample.com',
  'Mon-Sun 11:00-22:00',
  '["https://instagram.com/sample","https://facebook.com/sample"]',
  '["Brunch","Dinner","Delivery"]',
  '{"showGallery": true, "enableOrdering": true, "themeColor":"#c0392b", "seoTitle":"Sample Restaurant", "seoDesc":"Local, seasonal dishes"}'
);

INSERT INTO menus (restaurant_id, category, items_json) VALUES
(1, 'Appetizers', '[{"name":"Sample Salad","desc":"Fresh greens","price":10.00,"img":"/img/salad.jpg","available":true},{"name":"Garlic Bread","desc":"Toasted with herbs","price":5.50,"img":"/img/garlic.jpg","available":true}]'),
(1, 'Mains', '[{"name":"Grilled Fish","desc":"Daily catch","price":18.00,"img":"/img/fish.jpg","available":true},{"name":"Pasta","desc":"House sauce","price":15.00,"img":"/img/pasta.jpg","available":true}]');

INSERT INTO galleries (restaurant_id, images, captions) VALUES
(1, '["/img/hero.jpg","/img/dish1.jpg"]', '["Our entry","Signature dish"]');

INSERT INTO reviews (restaurant_id, testimonials) VALUES
(1, '[{"name":"John","rating":5,"comment":"Amazing food!","date":"2025-01-01T12:00:00Z"}]');
