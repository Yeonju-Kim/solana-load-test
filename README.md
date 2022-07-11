# solana-load-test

## Quick Start 
1. Build solanaslave/main.go 
```
$ cd solana-load-test
$ make build
```
2. Install locust in python3 virtual environment. 
```
$ python3 
$ source venv/bin/activate
(venv) $ pip3 install locust==1.2.3
```


3. Locust master: Run local master 
```
(venv)$ locust -f dist/locustfile.py --master 
```

4. Locust slave: Enter the next script with rich account key and RPC endpoints. 
```
./build/bin/solanaslave --max-rps 150 --master-host localhost --master-port 5557 -key ${KEY}  \
-tc="newValueTransferTx"  -endpoint ${ENDPOINT} -endpointWs ${WS_ENDPOINT}  -vusigned 20
```
