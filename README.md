itermctl
---

CLI interface and Go client library for iTerm2's API.

Client library coverage
===

- [RPC callbacks](https://pkg.go.dev/mrz.io/itermctl/pkg/itermctl/rpc?tab=doc#Callback) ([example](examples/rpc.go))
- [StatusBar components](https://pkg.go.dev/mrz.io/itermctl/pkg/itermctl/rpc?tab=doc#Callback) ([example](examples/statusbar.go))
- [CustomControlSequenceMonitor](https://pkg.go.dev/mrz.io/itermctl/pkg/itermctl?tab=doc#CustomControlSequenceMonitor),
  [NewSessionMonitor](https://pkg.go.dev/mrz.io/itermctl/pkg/itermctl?tab=doc#NewSessionMonitor),
  [TerminateSessionMonitor](https://pkg.go.dev/mrz.io/itermctl/pkg/itermctl?tab=doc#TerminateSessionMonitor),
  [PromptMonitor](https://pkg.go.dev/mrz.io/itermctl/pkg/itermctl?tab=doc#TerminateSessionMonitor) ([example](examples/rpc.go))
- Some methods to [work with windows, tabs and sessions](https://pkg.go.dev/mrz.io/itermctl/pkg/itermctl?tab=doc#NewApp)

CLI usage
===

### RPC commands

#### List sessions

```
$ itermctl rpc list-sessions --titles
a F982F64A-EBA0-4B94-95C4-1F445184CEE7 pty-924BF6D6-F780-4D0E-AD37-F1D6DC184E81 zsh
i 21243EC4-234D-433F-A8B0-0C47971184DC pty-924BF6D6-F780-4D0E-AD37-F1D6DC184E81 zsh
```

#### Sent text to a session

```
$ echo "ls -l" | itermctl rpc send-text F982F64A-EBA0-4B94-95C4-1F445184CEE7 
```

#### Split a session's pane

```
$ itermctl rpc split-pane F982F64A-EBA0-4B94-95C4-1F445184CEE7 
$ itermctl rpc split-pane --vertical F982F64A-EBA0-4B94-95C4-1F445184CEE7 
$ itermctl rpc split-pane --vertical --before F982F64A-EBA0-4B94-95C4-1F445184CEE7 
```

#### Create a tab in a window

```
$itermctl rpc create-tab pty-924BF6D6-F780-4D0E-AD37-F1D6DC184E81
```


### Add to AutoLaunch

Print to `stdout` a Python launcher script for a custom program, optionally saving it to the `AutoLaunch` directory
(with `--save-as`).

```
$ itermctl autolaunch --save-as=test -- go run my-component.go
from os import execv
execv("/usr/bin/go", ["go", "run", "my-component.go"])
saved to ~/Library/ApplicationSupport/iTerm2/Scripts/AutoLaunch/test.py
```
