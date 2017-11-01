#!/bin/bash
sqlite3 blocks.db 'DELETE FROM block'
sqlite3 blocks.db "INSERT INTO block (identifier,nonce,data,previous_hash,difficulty) VALUES	(\"000102030405060708090A0B0C0D0E0F\", 3754873684, \"Genesis Block for CTF contest, all block chains must start with this block. This is equivalent to Big Bang, time didn't exist before this\", \"None\", 8);"
sqlite3 blocks.db "INSERT INTO block (data,difficulty) VALUES (\"Four be the things I am wiser to know:\", 4)"
sqlite3 blocks.db "INSERT INTO block (data,difficulty) VALUES (\"Idleness, sorrow, a friend, and a foe.\", 4)"
sqlite3 blocks.db "INSERT INTO block (data,difficulty) VALUES (\"Four be the things I\'d been better without:\", 5)"
sqlite3 blocks.db "INSERT INTO block (data,difficulty) VALUES (\"Love, curiosity, freckles, and doubt.\", 6)"
sqlite3 blocks.db "INSERT INTO block (data,difficulty) VALUES (\"Three be the things I shall never attain:\", 7)"
sqlite3 blocks.db "INSERT INTO block (data,difficulty) VALUES (\"Envy, content, and sufficient champagne.\", 9)"
sqlite3 blocks.db "INSERT INTO block (data,difficulty) VALUES (\"Three be the things I shall have till I die:\", 11)"
sqlite3 blocks.db "INSERT INTO block (data,difficulty) VALUES (\"Laughter and hope and a sock in the eye.\", 13)"
sqlite3 blocks.db "INSERT INTO block (data,difficulty) VALUES (\" -Dorothy Parker\", 16)"
