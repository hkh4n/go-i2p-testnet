## Notes

This is in active development and not in working order yet.

## TO DO
 - [ ] Lifecycle startup + shutdown
 - [/] Volume for each router
 - [ ] Port forwarding
 - [/] Readline interface
 - [ ] Reseed via file
 - [ ] Reseed via node
 - [ ] go-i2p nodes
 - [ ] i2pd nodes
 - [ ] i2p java router nodes
 - Config
   - [X] go-i2p node
   - [ ] i2pd node
   - [ ] i2p java router node

## Verbosity ##
Logging can be enabled and configured using the DEBUG_I2P environment variable. By default, logging is disabled.

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