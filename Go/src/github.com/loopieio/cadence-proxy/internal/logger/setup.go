//-----------------------------------------------------------------------------
// FILE:		setup.go
// CONTRIBUTOR: John C Burnes
// COPYRIGHT:	Copyright (c) 2016-2019 by neonFORGE, LLC.  All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// SetLogger takes a log level and bool and sets
// the global zap.Logger to a custom configured logger where the properties
// are specified in the parameters
//
// param logLevel string -> the log level to set in the global logger
//
// param debugMode bool -> run in debug mode or not
func SetLogger(logLevel string, debugMode bool) {

	// new *zap.Logger
	// new zapcore.EncoderConfig for the logger
	var logger *zap.Logger
	var encoderCfg zapcore.EncoderConfig

	// new AtomicLevel for dynamic logging level
	atom := zap.NewAtomicLevel()

	switch debugMode {
	case true:

		// set the log level
		atom.SetLevel(zap.DebugLevel)

		// create the logger
		encoderCfg = zap.NewDevelopmentEncoderConfig()
		encoderCfg.TimeKey = "Time"
		encoderCfg.LevelKey = "Level"
		encoderCfg.MessageKey = "Debug Message"
		logger = zap.New(zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderCfg),
			zapcore.Lock(os.Stdout),
			atom,
		))
		defer logger.Sync()

	default:

		// set the log level
		switch logLevel {
		case "panic":
			atom.SetLevel(zap.PanicLevel)
		case "fatal":
			atom.SetLevel(zap.FatalLevel)
		case "error":
			atom.SetLevel(zap.ErrorLevel)
		case "warn":
			atom.SetLevel(zap.WarnLevel)
		case "debug":
			atom.SetLevel(zap.DebugLevel)
		default:
			atom.SetLevel(zap.InfoLevel)
		}

		// create the logger
		encoderCfg = zap.NewProductionEncoderConfig()
		encoderCfg.TimeKey = "Time"
		encoderCfg.MessageKey = "Message"
		encoderCfg.LevelKey = "Level"
		logger = zap.New(zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderCfg),
			zapcore.Lock(os.Stdout),
			atom,
		))
		defer logger.Sync()
	}

	// set the global logger
	_ = zap.ReplaceGlobals(logger)
}
