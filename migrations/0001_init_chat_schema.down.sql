-- Drop optional tables first (no dependencies)
DROP TABLE IF EXISTS message_throttle;
DROP TABLE IF EXISTS user_presence;

-- Drop messages and ordering infrastructure
DROP INDEX IF EXISTS idx_messages_conversation_seq;
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS conversation_sequences;

-- Drop conversation membership
DROP INDEX IF EXISTS idx_conversation_members_user;
DROP TABLE IF EXISTS conversation_members;

-- Drop conversations and users
DROP TABLE IF EXISTS conversations;
DROP TABLE IF EXISTS users;

-- Disable UUID extension (optional)
DROP EXTENSION IF EXISTS "pgcrypto";
