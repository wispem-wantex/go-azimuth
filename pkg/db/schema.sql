PRAGMA foreign_keys = on;

create table points (
	azimuth_number integer primary key, -- @p

	owner_address text not null default "" check (owner_address regexp "0x[0-9a-f]{40}"),
	owner_nonce integer not null default 0,
	spawn_address text not null default "" check (spawn_address regexp "0x[0-9a-f]{40}"),
	spawn_nonce integer not null default 0,
	management_address text not null default "" check (management_address regexp "0x[0-9a-f]{40}"),
	management_nonce integer not null default 0,
	voting_address text not null default "" check (voting_address regexp "0x[0-9a-f]{40}"),
	voting_nonce integer not null default 0,
	transfer_address text not null default "" check (transfer_address regexp "0x[0-9a-f]{40}"),
	transfer_nonce integer not null default 0,

	dominion integer not null default 1,
	is_active bool not null default 0,
	life integer not null default 0, -- How many times networking keys have been reset (starts at 1 on initializing keys)
	rift integer not null default 0, -- How many times the ship has breached (starts at 0)
	crypto_suite_version integer not null default 0, -- version of the crypto suite used for the pubkeys
	auth_key blob not null default "",  -- Authentication public key
	encryption_key blob not null default "", -- Encryption public key

	has_sponsor bool not null default 0, -- Don't want to deal with nullable ints in Go
	sponsor integer not null default 0, -- @p

	is_escape_requested bool not null default 0,
	escape_requested_to integer not null default 0 -- @p
);

create table operators (rowid integer primary key,
	owner_address text not null check (owner_address regexp "0x[0-9a-f]{40}"),
	authorized_operator_address text not null check (authorized_operator_address regexp "0x[0-9a-f]{40}"),

	unique (owner_address, authorized_operator_address)
);

create table dns (rowid integer primary key,
	text text
);

create table block_progress (rowid integer primary key,
	latest_block integer
);

create table event_logs (rowid integer primary key,
	block_number integer not null,
	block_hash text not null,
	tx_hash text not null,
	log_index integer not null,
	name text not null,
	topic0 text not null, topic1 text not null default "", topic2 text not null default "",
	data text not null default "",

	is_processed bool not null default 0,

	unique(block_number, log_index)
);


create table db_version (
	version integer
);
insert into db_version values(0);
