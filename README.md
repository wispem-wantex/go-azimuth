# Azimuth

A GAS stack based client for Azimuth.

For more on the GAS stack, read: https://twitter.com/wispem_wantex/status/1837350404301979951

Implemented as a subcommands program.  Available subcommands:

- get_logs_azimuth:
	Download Azimuth logs since the smart contract was launched.
- get_logs_naive:
	Download the Naive logs since the smart contract was launched.
- play_logs:
	Play (apply) the existing set of Azimuth logs already downloaded, up until the launch of the Naive contract
- play_logs_naive:
	Play (apply) logs, both Naive and Azimuth, since the launch of the Naive contract
- query:
	Once logs have been downloaded and played, you can query for points.
- show_logs:
	Once logs have been downloaded and played, you can show the historical event logs for a given point


## Infura / Ethereum node info and snapshotting

An Infura free account has a limit of 6 million "credits" per day.  Fetching the L1 is basically free; it takes about 30K-40K credits to fetch the whole thing, since it's all event logs that can be grouped in huge batches.  Fetching the L2 is far more expensive, costing about 400K credits.  This is because L2 data is stored in transaction call data, so the transactions have to be fetched by hash, one at a time.

Downloading the whole thing should still cost less than 10% of an Infura free tier daily credits quota.

For convenience, snapshots will be provided:

- event logs saved, but not played;
- event logs saved and played (i.e., full Azimuth state built)

These are optional; as long as you have an Ethereum node connection, you should be able to download the logs and rebuild the state from scratch in less than an hour.  Probably some people should do this periodically to make sure the snapshot is correct.  :)

## Building from scratch

Building the state from scratch requires an Ethereum node connection.  Each step should run in under 10 minutes.  To make it faster, see the section below (speedup).

After the two "get_logs_XXXX" commands, check that there's no panics (error stack traces) in the console.  If there are, *please send the whole output to me*!!  There's a hard-to-reproduce connection bug I'm trying to fix, it happens randomly like 1 in every 20 runs.  If a panic happens, the db file is invalid; just delete it start over (get the logs again, both L1 and L2).

```bash
# Compile it
go build -o azimuth ./cmd  # You don't have to call it `azimuth`, it's up to you

# Fetch the logs, L1 and L2.  You can pick any database filepath you want; I like `azimuth.db`.
azimuth --db azimuth.db get_logs_azimuth # L1
azimuth --db azimuth.db get_logs_naive   # L2

# Play the logs
azimuth --db azimuth.db play_logs_azimuth
azimuth --db azimuth.db play_logs_naive
```

### Tip: speedup with an in-memory database file

Since running the logs makes a lot of db reads and writes, you can speed it up quite a bit by running it on a temporary in-memory directory.  This will temporarily use a large block of memory; 500 MB should be enough.  Everything in that directory will be deleted when it's unmounted or you reboot, so copy the finished database file back to a normal directory once you're done.

You can do that like this:

```bash
# Create a directory to use as a memory-filesystem mount point
mkdir memory_dir

# Mount a tmpfs (temporary filesystem)
# `size=500M` should be enough
sudo mount -t tmpfs -o size=500M tmpfs memory_dir

# Run the commands from above
azimuth --db memory_dir/azimuth.db get_logs_azimuth
# ...etc

# Copy the database out of the memory filesystem so you don't lose it on reboot
cp memory_dir/azimuth.db .

# Unmount the tmpfs to get your memory back, once you're finished
sudo umount memory_dir
```

Using this trick will make the whole thing 8-10 times faster.
