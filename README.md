# SMS Server Alerts
### ⏱ 30 min build time

## Why build SMS server alerts?

For any online service advertising guaranteed uptime north of 99%, being
available and reliable is extremely important. Therefore it is essential
that any errors in the system are fixed as soon as possible. The
prerequisite for that is that error reports are delivered quickly to the
engineers on duty.
Providing these error logs via SMS ensures a faster response time compared
to email reports , helping  companies keep their uptime promises.

In this MessageBird Developer Guide, we will show you how to build an
integration of SMS alerts into a Go application that uses the
[Logrus](https://www.github.com/sirupsen/logrus) logging library.

## Logging with Go

Logging with Go comes with a few "gotchas". Typically, we send logs to the
terminal from our Go application using the `log` standard library. But
`log` has a few limitations:

- `log`, by default, timestamps your output and sends it to STDERR. To
change this, we have add the following line of code:

	```go
	log.SetOutput(os.Stdout)

	// You can also set your log output to a file:
	// file, err := os.OpenFile("tmp.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	// if err != nil {
	//   log.Println(err)
	// }
	// defer file.Close()
	// log.SetOutput(file)
	```

- `log` doesn't have built in log levels. Log levels allow us to specify a severity of a log message. For example, an **INFO** log level would be for informational logs, as opposed to an **ERROR** log level that would indicate an application error has occurred.
- `log` only lets us designate one output destination for each instance of a logger (`*log.Logger`). This means that to send a log event to two or more different outputs at a time, you would have to do something like this:
	```go
	LogThis := log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	LogThisToError := log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.LShortfile)
	file,_ := os.OpenFile("logfile.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	LogThisToFile := log.New(file, "ERROR: ", log.Ldate|log.Ltime|log.LShortfile)

	LogThis("I have to log the same output many times.")
	LogThisToError("I have to log the same output many times.")
	LogThisToFile("I have to log the same output many times.")
	```

While this is an entirely feasible way to set up logging in Go, we prefer
a simpler solution: using the [Logrus](https://github.com/sirupsen/logrus)
logging package.

### Introducing Logrus

[Logrus](https://github.com/sirupsen/logrus) is a Go logging package that:

- Has built in support for log levels. Instead of creating a new `log`
instance to manage each log level, levels are instead methods in `logrus`.
For example, to log an Error, we write `logrus.Error(err)`.
- The default logger can be extended with "hooks". Hooks allow us to
attach code to log events, which can be useful when we need extra logging
features like sending an SMS notification on certain log events.
- Is compatible with the standard library logger. Everything that you can do with the standard library logger, you can do with Logrus. This means that you can safely rewrite `log` with the Logrus logger with this import statement:

```go
import log "github.com/sirupsen/logrus"

func main(){
	log.Println("This line of code will run with the Logrus logger or the standard library logger.")
}
```

Logrus also has a library of hooks that you can use integrate your Go application's logging with a variety of other services at [https://github.com/sirupsen/logrus/wiki/Hooks](https://github.com/sirupsen/logrus/wiki/Hooks)

### Our application logging requirements

Our sample application is a web server that monitors its own health.
If it detects that a route produces a "server error" HTTP response status code
class (`5xx` status codes), our application:

- Displays the error in the terminal,
- Logs the error to a file,
- And sends an SMS notification to a designated recipient.

At the same time, we also want to continue sending informational logs to the terminal. But we don't want to write these logs to the log file, or trigger our application to send an SMS notification.

So, we want to customize our logger to be able to:

- Differentiate between log levels. Our logger must at least differentiate between error-level logs, and informational logs (info-level).
- Send logs to specific outputs according to their log levels.

To fulfill the above requirements, we'll be using the Logrus library and writing a custom hook that will send SMS notifications to a specified recipient only when our HTTP server encounters a category 5xx HTTP status code.

## Getting Started

Before we get started, let's make sure that you've installed the following:

- Go 1.11 and newer.
- [Logrus](https://github.com/sirupsen/logrus) 1.0.6 and newer.
- [MessageBird Go SDK](https://github.com/messagebird/go-rest-api) 5.0.0 and newer.

Now, let's install Logrus and the MessageBird Go SDK with the `go get` command:

```bash
go get -u -v github.com/sirupsen/logrus
go get -u -v github.com/messagebird/go-rest-api
```

You can find the sample code for this guide at the [MessageBird Developer Guides GitHub repository](https://github.com/messagebirdguides/lead-alerts-guide-go). Let's now download or clone the repository to view the example code and follow along with the guide.

### Create your API Key 🔑

To enable the MessageBird SDK, we need to provide an access key for the API. MessageBird provides keys in _live_ and _test_ modes. To get this application running, we will need to create and use a live API access key. Read more about the difference between test and live API keys [here](https://support.messagebird.com/hc/en-us/articles/360000670709-What-is-the-difference-between-a-live-key-and-a-test-key-).

Let's create your live API access key. First, go to the [MessageBird Dashboard](https://dashboard.messagebird.com/en/user/index); if you have already created an API key it will be shown right there. If you do not see any key on the dashboard or if you're unsure whether this key is in _live_ mode, go to the _Developers_ section and open the [API access (REST) tab](https://dashboard.messagebird.com/en/developers/access). Here, you can create new API keys and manage your existing ones.

If you are having any issues creating your API key, please reach out to our Customer Support team at support@messagebird.com.

**Pro-tip:** To keep our demonstration code simple, we will be saving our API key in `main.go`. However, hardcoding your credentials in the code is a risky practice that should never be used in production applications. A better method, also recommended by the [Twelve-Factor App Definition](https://12factor.net/), is to use environment variables. You can use open source packages such as [GoDotEnv](https://github.com/joho/godotenv) to read your API key from a `.env` file into your Go application. Your `.env` file should be written as follows:

`````env
MESSAGEBIRD_API_KEY=YOUR-API-KEY
`````

To use [GoDotEnv](https://github.com/joho/godotenv) in your application, let's type the following command to install it:

````bash
go get -u github.com/joho/godotenv
````

Then, let's import it in your application:

````go
import (
  // Other imported packages
  "os"

  "github.com/joho/godotenv"
)

func main(){
  // GoDotEnv loads any ".env" file located in the same directory as main.go
  err := godotenv.Load()
  if err != nil {
    log.Fatal("Error loading .env file")
  }

  // Store the value for the key "MESSAGEBIRD_API_KEY" in the loaded '.env' file.
  apikey := os.Getenv("MESSAGEBIRD_API_KEY")

  // The rest of your application ...
}
````

### Initialize the MessageBird Client

If you haven't already, let's now install the [MessageBird's REST API package for Go](https://github.com/messagebird/go-rest-api) by running:

````go
go get -u -v github.com/messagebird/go-rest-api
````

In your project folder which we created earlier, let's create a `main.go` file, and write the following code:

````go
package main

import (
  "os"

  "github.com/messagebird/go-rest-api"
)

func main(){
  client := messagebird.New("<enter-your-api-key>")
}
````

## Structuring our application

Now that we've initialized the MessageBird client, we can build our web server. Our web server:

- Doesn't need complex routing. We're just using it to demonstrate how to trigger an SMS notification on a server error, so we just need it have any rendered views display the HTTP status code the server responded with.
- Needs to encounter an error on demand. To do this, we'll build a route that takes a HTTP status code as part of the URL path, and renders a page for that status code and writes the appropriate HTTP response headers.

Once our web server code is written, we need to configure our logger by:

- Overwriting the standard library logger `log`.
- Writing a Logrus hook that allows us to select an output, and select which log events will be written to that output.
- Writing a custom MessageBird `type` that we can pass to our Logrus hook as an output destination.

### Building our web server

First, let's build a simple web server with a default route and a route that simulates a HTTP status code:

```go
package main

import (
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"
)

func defaultPath(w http.ResponseWriter, r *http.Request){
	fmt.Fprintln(w, "Hello. "+
    "Please enter a valid status code in the path to simulate a HTTP server status. "+
    "E.g. www.example.com/simulate/404")
}

func simulateHTTPStatus(w http.ResponseWriter, r *http.Request){
	fmt.Fprintln(w, "Hello. "+
    "Please enter a valid status code in the path to simulate a HTTP server status. "+
    "E.g. www.example.com/simulate/404")
}

func main(){
  client := messagebird.New("<enter-your-api-key>")

	http.HandleFunc("/",defaultPath)
	http.HandleFunc("/simulate/",simulateHTTPStatus)

	err := http.ListenAndServe(":8080",nil)
	if err != nil {
		log.Errorln(err)
	}
}
```

Notice that we're already using the Logrus logger in place of the standard library logger `log`. The default Logrus logger works well out-of-the-box, and allows us to add customizations progressively, so it makes sense to start using it early so we have our logging infrastructure in place while as we build our application.

We're also stubbing out two routes and their respective handlers: `simulateHTTPStatus` should simulate a HTTP status code on the `/simulate/` URL path, and `defaultPath` handles all other routes on our web server.

Next, we need to get our `simulateHTTPStatus` handler to read from the URL path and figure out which HTTP status code to simulate. Modify the `simulateHTTPStatus()` function to look like the following:

```go
func simulateHTTPStatus(w http.ResponseWriter, r *http.Request){
	path := strings.Split(r.URL.Path, "/")
	simulateCode, err := strconv.Atoi(path[2])
	if err != nil {
			log.Error(err)
		} else if len(path[2]) != 3 {
			output := fmt.Sprintf("Unknown status code used in path: %s", path[2])
			log.Warningln(output)
			fmt.Fprintln(w, output)
			fmt.Fprintln(w, "Hello. "+
				"Please enter a valid status code in the path to simulate a HTTP server status. "+
				"E.g. www.example.com/simulate/404")
			return
		}
}
```

Here, we're setting up a route where entering `www.example.com/simulate/500` would tell our server to simulate a 500 HTTP status code. To do this, we get the URL path with `r.URL.Path` and split it into sections delimited by `/`. Because we expect a fixed URL path format of `www.example.com/simulate/<StatusCode>`, we can safely set our `simulateCode` variable to `path[2]`.

If the user enters an unexpected URL path, we handle it either in `simulateHTTPStatus` itself or fall back to the `defaultPath` handler. Both render a page that tells the user to enter a path like "www.exampe.com/simulate/404".

Once we're sure that we're getting a URL path that contains a HTTP status code that we can simulate, we can write the logic to handle it. Add the following code to the bottom of `simulateHTTPStatus()`:

```go
func simulateHTTPStatus(w http.ResponseWriter, r *http.Request){
	// ... previous code
	w.WriteHeader(simulateCode)

	// Handle all possible Server Error class of HTTP status codes
	if simulateCode >= 500 && simulateCode < 600 {
		output := fmt.Sprintf("Server error. [%s %s] %d %s", r.Method, r.URL.Path, simulateCode, http.StatusText(simulateCode))
		log.Errorln(output)
		fmt.Fprintln(w, output)
	} else {
		output := fmt.Sprintf("Everything's ok on our end.[%s %s] %d %s", r.Method, r.URL.Path, simulateCode, http.StatusText(simulateCode))
		log.Infoln(output)
		fmt.Fprintln(w, output)
	}
	return
}
```

Above, we're writing our status code `simulateCode` to our HTTP response header, and then checking if it is a category 5xx status code. If it's a category 5xx status code, we log an error. If it's not a category 5xx status code, then we log it as an informational log event.

Now we've got all our web server logic set up, we can move on to writing our custom Logrus hooks to handle error logging according to our requirements.

### Building our custom Logrus hook

To get to a point where our logger can send SMS notifications when an error is logged, we need to first write a custom hook that writes to a given output for the log levels we specify.

**Info**: In this guide, we won't write a dedicated hook like those found in the Logrus [list of hooks](https://github.com/sirupsen/logrus/wiki/Hooks). Instead, we'll
write a generic hook that allows us to peek at how Logrus hooks work, and allows
us to quickly change logging behaviour from within our application itself.

Add the following code under your `main()` block:

```go
// WriterHook hooks into log events.
type WriterHook struct {
  Writer    io.Writer
  LogLevels []log.Level
}

// Fire tells WriterHook what to do when an event is logged.
func (hook *WriterHook) Fire(entry *log.Entry) error {
  line, err := entry.String()
  if err != nil {
    return err
  }
  _, err = hook.Writer.Write([]byte(line))
	if err != nil {
		return err
	}
  return nil
}

// Levels rewrites the Levels method to only include the
// log.Level specified in the WriterHook struct.
func (hook *WriterHook) Levels() []log.Level {
  return hook.LogLevels
}
```

Here, we've:

- Added a new struct type named `WriterHook`. `WriterHook` takes two parameters, an `io.Writer` and a list of log levels, that we'll use when writing our `Fire()` implementation.
- Implemented a `Fire()` method for `WriterHook`. When a log event occurs,
Logrus calls `Fire()` on all hooks attached to a logger
and passes the contents of that log event (`entry *log.Entry`) into it.
In our `Fire()` implementation, we parse the contents of the log event `entry`,
and then send it to the `io.Writer` we've attached to our hook.
- Implemented a `Levels()` method for `WriterHook` that just returns the list of
levels we specify when initializing the `WriterHook` struct.
Logrus reads a list of levels from a hook's `Levels()` method,
and only triggers that hook for the levels contained within this list.

**Note**: Don't try to log messages within hooks — if you're not using a separate logger instance, your application will be sent into an infinite loop. Instead, make sure that you handle all errors by returning them for either the Logrus library or the main application to handle.

Once we've done this, we can add the hooks to our logger.

First, let's declare a few lists of log levels to help us keep our hook definitions brief:

```go
var (
  logLevelsInfo = []log.Level{
    log.InfoLevel,
    log.WarnLevel,
  }
  logLevelsSevere = []log.Level{
    log.ErrorLevel,
    log.PanicLevel,
    log.FatalLevel,
  }
  logLevelsAll = []log.Level{
    log.InfoLevel,
    log.WarnLevel,
    log.ErrorLevel,
    log.PanicLevel,
    log.FatalLevel,
  }
)
```

Then, modify `main()` to look like the following:

```go
func main(){
	client := // ...

	log.SetOutput(ioutil.Discard)
	log.SetFormatter(&log.JSONFormatter{})

	logfile, err := os.OpenFile("mbservermon.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Error(err)
	}
	defer func() {
		logfile.Sync()
		logfile.Close()
	}()

	log.AddHook(&WriterHook{
		Writer:    os.Stdout,
		LogLevels: logLevelsAll,
	})
	log.AddHook(&WriterHook{
		Writer:    logfile,
		LogLevels: logLevelsAll,
	})

	// ...
}
```

In the code snippet above, we:

- Define three lists of log levels, which we will use to write our hooks.
- Discard default logging output for our logger `log`,
as we're delegating all log output to the hooks we'll attach to the it.
- Set the logger's output format to JSON.
This gives us log output that looks like:

	```JSON
	{"level":"info","msg":"Serving on:1313","time":"2018-09-20T13:11:01+08:00"}
	```
	You can also set the log output format to `&log.TextFormatter{}`, which gives us
	log output that looks like:

	```
	time="2018-09-21T01:01:01+08:00" level=info msg="Serving on:1313"
	```

- Initialize our log file as `logfile`, which we will write logs to.
- Once we've done all of the above, we then write two hooks. The first hook tells
our logger to send log events from all levels to the terminal as STDOUT.
The second hook tells our logger to write log events from all levels to `logfile`.

### Writing a custom `io.Writer` that sends SMS notifications

Now that we've got our basic logging functionality set up,
we can start writing code to send SMS notifications when our web server
encounters an error.

First, we need to set our code up so that we can write to the MessageBird
REST API like it's a file. To do this, we'll write a `MBContainer` struct type
that we attach a `Write()` method to so that it qualifies as an `io.Writer`.

We've also added Client, Originator, and Recipient parameters to the `MBContainer`
struct so that we can configure these from `main()` when writing our hook.

```go
type MBContainer struct {
	Client     *messagebird.Client
	Originator string
	Recipients []string
}

func (mb *MBContainer) Write(p []byte) (int, error) {
	msgBody := string(p[:])
	if len(msgBody) > 160 {
		msgBody = msgBody[:159]
	}
	msg, err := sms.Create(
		mb.Client,
		mb.Originator,
		mb.Recipients,
		msgBody,
		nil,
	)
	if err != nil {
		return 1, err
	}
	fmt.Println("Message sent: %v", msg)
	return 0, nil
}
```

In this `Write()` method:

- We take the data written to `Write()` and turn it into a `msgBody` string
- Then, we check that `msgBody` is not more than 160 characters,
or MessageBird will split the log message into two SMS notifications.
- We then call `sms.Create()` which tells the MessageBird REST API
to send it as an SMS notification.

Once this is done, add a new hook to `main()`:

```go
func main(){
	// ...
	log.AddHook(&WriterHook{
		Writer:    os.Stdout,
		LogLevels: logLevelsAll,
	})
	log.AddHook(&WriterHook{
		Writer:    logfile,
		LogLevels: logLevelsAll,
	})

	// MessageBird hook
	log.AddHook(&WriterHook{
		Writer:    &MBContainer{client, "MBServerMon", []string{"<recipient_number_here>"}},
		LogLevels: logLevelsSevere,
	})

	// ...
}
```

Here, we've written a new hook that takes an anonymous struct `&MBContainer{}`
as the output to write to, and logs only "severe" log events to this output.
We've earlier defined "severe" log events to be of log levels `log.ErrorLevel`,
`log.PanicLevel`, and `log.FatalLevel`.

In `&MBContainer{}`, we pass in our
MessageBird `client`, an "originator" `MBServerMon`
(which has to be a string with a maximum of 11 characters),
and a list of recipients for the log messages.

**Note**: Remember to replace `<recipient_number_here>` in your code with the
phone number of the recipient who needs to receive server error notifications,
written in an international format (e.g. "+319876543210").

That's all you need to do! You've now set up leveled logging for your web server
application that can send specific log levels to set outputs. More
importantly, if your web server application encounters an error, it sends an
SMS notification to your system administrator to tell them to get to work on it!

## Testing the Application

To run the application, go to your terminal and run:

````bash
go run main.go
````

Next, navigate to http://localhost:8080.

You should see the following message displayed:

```
Hello. Please enter a valid status code in the path to simulate a HTTP server status. E.g. www.example.com/simulate/404
```

Then, navigate to http://localhost:8080/simulate/404.
This should trigger a warning log event that is displayed in your terminal and
logged to your log file, but does not trigger an SMS notification.

Finally, navigate to http://localhost:8080/simulate/500.
This should trigger a warning log event that is displayed in your terminal and
logged to your log file. In addition, it should also send an SMS notification
with the error message.

## Nice work!

And that's it. You now have a working Go web application that simulates HTTP status
codes, and an implementation of the Logrus logger that can send SMS notifications
on server error events.

You can now take these elements and integrate them into a Go production application.
Don't forget to download the code from the [MessageBird Developer Guides GitHub repository](https://github.com/messagebirdguides/sms-server-alerts-guide-go).

## Next steps

Want to build something similar but not quite sure how to get started? Please feel free to let us know at support@messagebird.com, we'd love to help!
