# Azimuth

A GAS stack based client for Azimuth.

For more on the GAS stack, read: https://twitter.com/wispem_wantex/status/1837350404301979951

Implemented as a subcommands program.  Available subcommands:

- get_logs:
	Download Azimuth logs since the smart contract was launched.  They can be applied in-place with the `--apply` flag, or without applying to create a snapshot of just the events
- play_logs:
	Play (apply) the existing set of logs already downloaded
- query:
	Once logs have been downloaded and played, you can query for points.


## Azimuth L2 / "Naive Rollup"

Not implemented yet.  Soon(tm).  (But actually soon, though.)
