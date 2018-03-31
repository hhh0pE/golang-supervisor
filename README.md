# golang-supervisor
Small tool to automate start small background process that look for main process and launch new one if there is a crash

# Usage
just import this package on top of your application imports, like this:

    import (
 	_ "github.com/hhh0pE/golang-supervisor"
 	"log"
 	"os"
 	"strings"
 	"io"
 	"os/exec"
 	"flag"
 	"fmt"
 	"time"
    )

About other takes care golang-supervisor. It will create 2 copy of source executable: [exeName].supervisor and [exeName].running and.

Supervisor works simple: 

if .running executable fails with code different from 0(zero means everything is okey), supervisor will restart .running executable immediately (but supervisor will copy executable from [exeName] to [exeName].running again, so it can be use for easily restart app).

Also you can get original executable name(without .running and .supervisor suffix) with method golang_supervisor.OriginalExecutablePath().  