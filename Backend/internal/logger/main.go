package logger 

import(
	"io"
	"os"
	"log/slog"
	"time"
	"github.com/lestrrat-go/file-rotatelogs"
)

//Global logger will be initialized in main
var Logger *slog.Logger

func InitLogger(env string){
// Create logs directory automatically
  if err := os.MkdirAll("logs", 0755); err != nil {
      panic("Failed to create logs directory: " + err.Error())
  }

	//initiates the writer for rotateLogs 
	writer, err := rotatelogs.New(
		"logs/app-%Y-%m-%d.log",                 // ← EXACTLY this string
		rotatelogs.WithLinkName("logs/app.log"), // symlink latest → app.log
		rotatelogs.WithMaxAge(30*24*time.Hour),  // keep 30 days
		rotatelogs.WithRotationTime(24*time.Hour), // rotate at midnight
	)
	if err != nil {
		panic("Failed to create rotatelogs: " + err.Error())
	}

	//json type log store in prod which helps other logging tools
	var handler slog.Handler
	if env == "production"{
		handler = slog.NewJSONHandler(writer, &slog.HandlerOptions{
			Level: slog.LevelInfo, // Change to Debug in dev
		})
	} else {  //for dev text type log store
		handler = slog.NewTextHandler(io.MultiWriter(os.Stdout, writer), &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	}
	Logger = slog.New(handler)
	slog.SetDefault(Logger)
}
