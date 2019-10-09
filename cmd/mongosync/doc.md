sync all transactions
```
./build/bin/mongosync --attach [ipc socket path] -s [start block number]
```
sync transactions related to selected addresses
```
./build/bin/mongosync --attach [ipc socket path] -s [start block number] -p true --addresses [addr1, addr2, ...]
```
