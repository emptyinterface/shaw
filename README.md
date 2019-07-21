#shaw
farming clouds

ssh remote command exection, file transfer, and light stream processing.  light abstraction on existing cloud provider libraries to unify resource management.

* run - throttled
* execute - parallel
* scan - parallel
* put - parallel
* put\_file - parallel/throttled - file per proc
* put\_file\_recursive - parallel/throttled - walk per proc
* rsync - throttled - until suitable go option
* local->remote pipe - parallel (since local stdout->stdin)

all commands support transparent gzipped io

parallel actions can consume a single stream and emit multiple identical streams.  ie. read from single file stream and output to the stdin of 4 concurrent ssh commands.  but parallel actions can only consume the stream once.  so they must act on all servers simultaneously.  under certain thresholds this is preferrable and ostensibly faster.

throttled actions require the stream to be scoped to the proc.  ie. set up a multiple file streams of the same file, one for each proc.  this duplicates read()s, one per proc.

parallel actions read once and write n times.  throttled actions read n times and write n times.

## actions
* deploy binary - runs a build/test/push sequence
* daisy chain deploy - sets up fifoqueues piped into tee piped into the next server in the list.  the chain begins by writing to the first server. it ends, optionally with a md5 checksum verification on each server.