PRAGMA foreign_keys = on;

create table points (
	azimuth_number integer primary key, -- @p

	owner_address blob not null default X'0000000000000000000000000000000000000000' check (length(owner_address) = 20),
	owner_nonce integer not null default 0,
	spawn_address blob not null default X'0000000000000000000000000000000000000000' check (length(spawn_address) = 20),
	spawn_nonce integer not null default 0,
	management_address blob not null default X'0000000000000000000000000000000000000000' check (length(management_address) = 20),
	management_nonce integer not null default 0,
	voting_address blob not null default X'0000000000000000000000000000000000000000' check (length(voting_address) = 20),
	voting_nonce integer not null default 0,
	transfer_address blob not null default X'0000000000000000000000000000000000000000' check (length(transfer_address) = 20),
	transfer_nonce integer not null default 0,

	dominion integer not null default 1,
	is_active bool not null default 0,
	life integer not null default 0, -- How many times networking keys have been reset (starts at 1 on initializing keys)
	rift integer not null default 0, -- How many times the ship has breached (starts at 0)
	crypto_suite_version integer not null default 0, -- version of the crypto suite used for the pubkeys
	auth_key blob not null default X'',  -- Authentication public key
	encryption_key blob not null default X'', -- Encryption public key

	has_sponsor bool not null default 0, -- Don't want to deal with nullable ints in Go
	sponsor integer not null default 0, -- @p

	is_escape_requested bool not null default 0,
	escape_requested_to integer not null default 0 -- @p
);
create view readable_points as
	select
    azimuth_number,
    lower(hex(owner_address)) as owner_address,
    owner_nonce,
    lower(hex(spawn_address)) as spawn_address,
    spawn_nonce,
    lower(hex(management_address)) as management_address,
    management_nonce,
    lower(hex(voting_address)) as voting_address,
    voting_nonce,
    lower(hex(transfer_address)) as transfer_address,
    transfer_nonce,
    dominion,
    is_active,
    life,
    rift,
    crypto_suite_version,
    lower(hex(auth_key)) as auth_key,
    lower(hex(encryption_key)) as encryption_key,
    has_sponsor,
    sponsor,
    is_escape_requested,
    escape_requested_to
from points;

create table operators (rowid integer primary key,
	owner_address blob not null check (length(owner_address) = 20),
	authorized_operator_address blob not null check (length(authorized_operator_address) = 20),

	unique (owner_address, authorized_operator_address)
);

create table dns (rowid integer primary key,
	text text
);

create table block_progress (rowid integer primary key,
	contract_address blob not null unique collate nocase,
	latest_block integer
);

create table event_types (rowid integer primary key,
	contract_address blob not null collate nocase check (length(contract_address) = 20),
	hashed_name blob unique not null,
	name text not null,
	unique (contract_address, hashed_name)
);
create view readable_event_types as
	select "0x" || lower(hex(contract_address)), lower(hex(hashed_name)), name from event_types; -- write the blob in hex format

create table event_logs (rowid integer primary key,
	block_number integer not null,
	block_hash blob not null,
	tx_hash blob not null,
	log_index integer not null,

	contract_address blob not null collate nocase,
	topic0 blob not null,
	topic1 blob not null default "",
	topic2 blob not null default "",
	data blob not null default "",

	is_processed bool not null default 0,

	unique(block_number, log_index)
	foreign key(contract_address, topic0) references event_types(contract_address, hashed_name)
);
create index index_event_logs_is_processed on event_logs(is_processed);
create view readable_event_logs as
	select block_number, lower(hex(block_hash)) hex_block_hash, lower(hex(tx_hash)) hex_tx_hash, log_index, "0x" || lower(hex(event_logs.contract_address)) hex_contract_address, name, lower(hex(topic0)) hex_topic0, lower(hex(topic1)) hex_topic1, lower(hex(topic2)) hex_topic2, lower(hex(data)) hex_data, is_processed from event_logs join event_types on topic0 = hashed_name;

create table db_version (
	version integer
);
insert into db_version values(0);
