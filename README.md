# alsa

A pure Go audio binding to ALSA that supports audio playback.

[![Build Status](https://travis-ci.org/anisse/alsa.svg?branch=master)](https://travis-ci.org/anisse/alsa)
[![Go Report Card](https://goreportcard.com/badge/github.com/anisse/alsa)](https://goreportcard.com/report/github.com/anisse/alsa)
[![GoDoc](https://godoc.org/github.com/anisse/alsa?status.svg)](http://godoc.org/github.com/anisse/alsa)

## WARNING

The public API is unstable. Please fork/vendor before using. Right now it's inspired by [oto](https://github.com/hajimehoshi/oto), but only the player part is implemented, and some features are missing that alsa-lib might do to make it plug-and-play:

 - format conversion
 - channel conversion
 - resampling

## TODO

This library implements the bare minimum of the rich alsa API. What's missing:

 - hardware feature detection (sample rate, channels, formats)
 - capture
 - non-interleaved audio playback
 - mmap-based API (less copying)
 - others
