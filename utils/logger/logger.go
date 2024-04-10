package zhifou

import (
  "os"
  "time"

  "go.uber.org/zap"
  "go.uber.org/zap/zapcore"
)

var zapLog *zap.Logger


const (
  logTmFmt = "2006-01-02 15:04:05"
)

func init() {
  Encoder := GetEncoder()
  newCore := zapcore.NewTee(
    zapcore.NewCore(Encoder, zapcore.Lock(os.Stdout), zapcore.DebugLevel), // 写入控制台
  )
  zapLog = zap.New(newCore, zap.AddCaller(), zap.AddCallerSkip(1))
}

// GetEncoder 自定义的Encoder
func GetEncoder() zapcore.Encoder {
  return zapcore.NewConsoleEncoder(
    zapcore.EncoderConfig{
      TimeKey:           "ts",
      LevelKey:          "level",
      NameKey:           "logger",
      CallerKey:         "caller",
      ConsoleSeparator:  " ",
      FunctionKey:       zapcore.OmitKey,
      MessageKey:        "msg",
      StacktraceKey:     "stacktrace",
      LineEnding:        "\n",
      EncodeLevel:       cEncodeLevel,
      EncodeTime:        cEncodeTime,
      EncodeDuration:    zapcore.SecondsDurationEncoder,
      EncodeCaller:      cEncodeCaller,
    })
}

// GetConsoleEncoder 输出日志到控制台
func GetConsoleEncoder() zapcore.Encoder {
  return zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
}


func Info(message string, fields ...zap.Field) {
  zapLog.Info(message, fields...)
}

func Debug(message string, fields ...zap.Field) {
  zapLog.Debug(message, fields...)
}

func Error(message string, fields ...zap.Field) {
  zapLog.Error(message, fields...)
}

func Fatal(message string, fields ...zap.Field) {
  zapLog.Fatal(message, fields...)
}

func Warn(message string, fields ...zap.Field) {
  zapLog.Warn(message, fields...)
}

func GetLevelEnabler() zapcore.Level {
  return zapcore.InfoLevel
}

func cEncodeLevel(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
  enc.AppendString("[" + level.CapitalString() + "]")
}

func cEncodeTime(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
  enc.AppendString("[" + t.Format(logTmFmt) + "]")
}

func cEncodeCaller(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
  enc.AppendString("[" + caller.TrimmedPath() + "]")
}