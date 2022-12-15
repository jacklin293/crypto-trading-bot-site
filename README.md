# DEPRECATED

This is an experimental repo and no longer maintained.

# Set up the environment

Run this command

```
docker-compose up -d
```
# Deploy

    make deploy

# Test cases

* close position via website
    * -> no error
* close position and cancel stop-loss order (via FTX app) -> close position via website
    * -> error, can't be closed
* close position by StopLossTriggered by engine
    * -> no error
* close position and cancel stop-loss order (via FTX app) -> trigger stop-loss by engine
    * -> no error, but say it's closed before
* close position and cancel stop-loss order (via FTX app) -> trigger take-profit by engine
    * -> error: out of sync
* close position by TakeProfitTriggered by engine
    * -> no error
