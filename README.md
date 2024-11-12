## Notes

This is in active development and not in working order yet. It is highly unstable and will fluctuate in changes.

## TO DO
 - [X] Lifecycle startup + shutdown
 - [X] Volume for each router
 - [X] Shared Volume
 - [ ] Port forwarding
 - [X] Readline interface
 - [ ] Reseed via file
 - [ ] Reseed via node (i2pd)
 - [ ] Reseed via node (i2p java)
 - [X] go-i2p node (basic startup)
 - [X] i2pd node (basic startup)
 - [ ] i2p java router node (basic startup)
 - [ ] Force netdb synchronization
 - Config
   - [X] go-i2p node
   - [X] i2pd node
   - [ ] i2p java router node
 - Metrics
   - [ ] TCP connection with daemon to relay router information

## Verbosity ##
Logging can be enabled and configured using the DEBUG_TESTNET environment variable. By default, logging is disabled.

There are three available log levels:

- Debug
```shell
export DEBUG_TESTNET=debug
```
- Warn
```shell
export DEBUG_TESTNET=warn
```
- Error
```shell
export DEBUG_TESTNET=error
```

If DEBUG_TESTNET is set to an unrecognized variable, it will fall back to "debug". Note, that this only accounts for verbosity in the testnet program itself, and not the nodes.