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

-- =======================
-- Ethereum tracking state
-- =======================

------
-- Contracts that we are tracking
create table contracts (rowid integer primary key,
	address blob not null unique collate nocase check (length(address) = 20),
	name text not null,
	start_block integer not null,
	latest_block_fetched integer not null default 0
);
insert into contracts (address, name, start_block) values
	(X'223c067f8cf28ae173ee5cafea60ca44c335fecb', 'Azimuth', 6784880),
	(X'eb70029cfb3c53c778eaf68cd28de725390a1fe9', 'Naive', 13369829);

------
-- Valid event types from the contracts we are tracking
create table event_types (rowid integer primary key,
	contract_address blob not null collate nocase check (length(contract_address) = 20),
	hashed_name blob unique not null,
	name text not null,

	unique (contract_address, hashed_name)
	foreign key(contract_address) references contracts(address)
);
create view readable_event_types as  -- write the blob in hex format
	select "0x" || lower(hex(contract_address)), lower(hex(hashed_name)), name from event_types;
insert into event_types (contract_address,hashed_name,name) values
	(X'223c067f8cf28ae173ee5cafea60ca44c335fecb',X'e74c03809d0769e1b1f706cc8414258cd1f3b6fe020cd15d0165c210ba503a0f','Activated'),
	(X'223c067f8cf28ae173ee5cafea60ca44c335fecb',X'b2d3a6e7a339f5c8ff96265e2f03a010a8541070f3744a247090964415081546','Spawned'),
	(X'223c067f8cf28ae173ee5cafea60ca44c335fecb',X'd7704f9a25193dbd0b0cb4a809feffffa7f19d1aae8817a71346c194448210d5','LostSponsor'),
	(X'223c067f8cf28ae173ee5cafea60ca44c335fecb',X'29294799f1c21a37ef838e15f79dd91bcee2df99d63cd1c18ac968b129514e6e','BrokeContinuity'),
	(X'223c067f8cf28ae173ee5cafea60ca44c335fecb',X'16d0f539d49c6cad822b767a9445bfb1cf7ea6f2a6c2b120a7ea4cc7660d8fda','OwnerChanged'),
	(X'223c067f8cf28ae173ee5cafea60ca44c335fecb',X'902736af7b3cefe10d9e840aed0d687e35c84095122b25051a20ead8866f006d','ChangedSpawnProxy'),
	(X'223c067f8cf28ae173ee5cafea60ca44c335fecb',X'cbd6269ec71457f2c7b1a22774f246f6c5a2eae3795ed7300db517680c61c805','ChangedVotingProxy'),
	(X'223c067f8cf28ae173ee5cafea60ca44c335fecb',X'b4d4850b8f218218141c5665cba379e53e9bb015b51e8d934be70210aead874a','EscapeRequested'),
	(X'223c067f8cf28ae173ee5cafea60ca44c335fecb',X'd653bb0e0bb7ce8393e624d98fbf17cda5902c8328ed0cd09988f36890d9932a','EscapeCanceled'),
	(X'223c067f8cf28ae173ee5cafea60ca44c335fecb',X'7e447c9b1bda4b174b0796e100bf7f34ebf36dbb7fe665490b1bfce6246a9da5','EscapeAccepted'),
	(X'223c067f8cf28ae173ee5cafea60ca44c335fecb',X'aa10e7a0117d4323f1d99d630ec169bebb3a988e895770e351987e01ff5423d5','ChangedKeys'),
	(X'223c067f8cf28ae173ee5cafea60ca44c335fecb',X'ab9c9327cffd2acc168fafedbe06139f5f55cb84c761df05e0511c251e2ee9bf','ChangedManagementProxy'),
	(X'223c067f8cf28ae173ee5cafea60ca44c335fecb',X'cfe369b7197e7f0cf06793ae2472a9b13583fecbed2f78dfa14d1f10796b847c','ChangedTransferProxy'),
	(X'223c067f8cf28ae173ee5cafea60ca44c335fecb',X'fafd04ade1daae2e1fdb0fc1cc6a899fd424063ed5c92120e67e073053b94898','ChangedDns'),
	(X'eb70029cfb3c53c778eaf68cd28de725390a1fe9',X'cca739c72762deed05941b38d4aa82f2718c74457d5e2d8c5b1d7642caf22196','Batch');

------
-- Ethereum event logs
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
