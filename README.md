# Azimuth

A GAS stack based client for Azimuth.

For more on the GAS stack, read: https://twitter.com/wispem_wantex/status/1837350404301979951

Implemented as a subcommands program.  Available subcommands:

- get_logs_azimuth:
	Download Azimuth logs since the smart contract was launched.  They can be applied in-place with the `--apply` flag, or without applying to create a snapshot of just the events
- get_logs_naive:
	Download the Naive logs since the smart contract was launched.
- play_logs:
	Play (apply) the existing set of Azimuth logs already downloaded, up until the launch of the Naive contract
- play_logs_naive:
	Play (apply) logs, both Naive and Azimuth, since the launch of the Naive contract
- query:
	Once logs have been downloaded and played, you can query for points.


## Infura / Ethereum node info and snapshotting

An Infura free account has a limit of 6 million "credits" per day.  Fetching the L1 is basically free; it takes about 30K-40K credits to fetch the whole thing, since it's all event logs that can be grouped in huge batches.  Fetching the L2 is far more expensive, costing about 400K credits.  This is because L2 data is stored in transaction call data, so the transactions have to be fetched by hash, one at a time.

Downloading the whole thing should still cost less than 10% of an Infura free tier daily credits quota.

For convenience, snapshots will be provided:

- event logs saved, but not played;
- event logs saved and played (i.e., full Azimuth state built)

These are optional; as long as you have an Ethereum node connection, you should be able to download the logs and rebuild the state from scratch in less than an hour.  Probably some people should do this periodically to make sure the snapshot is correct.  :)
