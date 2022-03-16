# How to run integration tests for 11-Beefy
<br/>

### First run the relay chain & parachain nodes:

<br/>

```bash
docker run -ti -p9944:9944 -p9988:9988 -p9998:9998 composablefi/composable-sandbox:latest
```
<br/>

### Then wait until you see this line: 
<br/>

```
🚀 POLKADOT LAUNCH COMPLETE 🚀
```
<br/>

### Now you can run the tests in `./11-beefy/types/update_test.go`
<br/>

```bash
go test -test.timeout=0 -run TestCheckHeaderAndUpdateState -v
```
<br/>

### You should start to see these lines:
<br/>

```bash
==== connected! ==== 
====== subcribed! ======


Initializing client state


clientState.LatestBeefyHeight: 169
clientState.MmrRootHash: 89a2850e8b5e475980ca1ef4c145f4c5624a072d287b85f0430815d5c9b7b387
====== successfully processed justification! ======
```

### This means the light client is following the relay chain consensus protocol