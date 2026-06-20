ALTER TABLE links ADD COLUMN user_id UUID REFERENCES users(id) ON DELETE CASCADE;
CREATE INDEX idx_links_user_id ON links(user_id);
