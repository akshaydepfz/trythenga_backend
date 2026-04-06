CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS restaurants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    owner_name TEXT NOT NULL,
    phone TEXT NOT NULL,
    email TEXT NOT NULL,
    address TEXT NOT NULL,
    city TEXT NOT NULL,
    state TEXT NOT NULL,
    pincode TEXT NOT NULL,
    country TEXT NOT NULL,
    gst_number TEXT NOT NULL,
    logo_url TEXT NOT NULL,
    opening_time TEXT NOT NULL,
    closing_time TEXT NOT NULL,
    seating_capacity INT NOT NULL DEFAULT 0,
    plan TEXT NOT NULL DEFAULT 'free' CHECK (plan IN ('free', 'premium')),
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_restaurants_name ON restaurants (name);
CREATE INDEX IF NOT EXISTS idx_restaurants_status ON restaurants (status);
