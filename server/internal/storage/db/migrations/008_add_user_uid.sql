-- Add uid column to track agent ownership
-- human users (type=0): uid = their own id
-- agent users (type=1): uid = creator's id
ALTER TABLE users ADD COLUMN IF NOT EXISTS uid TEXT NOT NULL DEFAULT '';
UPDATE users SET uid = id WHERE type = 0 AND uid = '';
