![Release][latest-release]
![Release Date][release-date]
![Language][language]
![License][license]
![Code Size][code-size]

<!-- PROJECT LOGO -->
<br />
<p align="center">
  <a href="https://github.com/demon-rem/auto-sub/">
    <img src="./assets/logo.png" alt="Logo" width="320" height="160">
  </a>

  <h3 align="center">auto-sub</h3>

  <p align="center">
    A command-line utility to batch-add subtitles to media files
    <br><br>
    <a href="https://github.com/demon-rem/auto-sub/"><strong>Explore the docs »</strong></a>
    <br />
    <br />
    <a href="https://github.com/demon-rem/auto-sub/issues">Bug Report</a>
    ·
    <a href="https://github.com/demon-rem/auto-sub/issues">Request a Feature</a>
    ·
    <a href="https://github.com/demon-rem/auto-sub/fork">Fork Repo</a>

  </p>
</p>
<br>

---
<br>

<!-- TABLE OF CONTENTS -->
- [About the project](#about-the-project)
- [Terminology](#terminology)
    - [Extra File](#extra-file)
    - [Media File](#media-file)
    - [Source Directory](#source-directory)
    - [Root Directory](#root-directory)
- [Installation](#installation)
  - [Compiling from source](#compiling-from-source)
- [Setup](#setup)
- [Documentation](#documentation)
    - [Syntax](#syntax)
  - [Flags](#flags)
  - [Boolean Flags](#boolean-flags)
  - [Miscellaneous Flags](#miscellaneous-flags)
- [License](#license)
- [Roadmap](#roadmap)

## About the project

A command line tool to batch add subtitles, chapters, attachments to media files using [FFmpeg](http://ffmpeg.org).

The final result will be present inside a matroska (`.mkv`) container.

## Terminology

#### Extra File
Extra file refers to any non-media file type. This can be subtitles, chapters, attachments, tags, etc.

#### Media File
Media file is the main input file for FFmpeg, can be a file of type `.mkv`, `.mp4`, etc.

#### Source Directory 
The main directory containing an individual media file. Each source directory should contain exactly one media file (`mkv/mp4/webm/etc`), and atleast one or more subtitle file, attachments or chapters.

Requirements for a valid source directory are:
 - Contains exactly one [media file](#media-file)
 - Contains atleast one or more [extra file](#extra-file) (extra files can be subtitles, chapters, attachments, etc)

#### Root Directory
Parent directory containing atleast one or more source directories.

The only requirements for a valid root directory are; it should contain one or more source directories.

As an example;
```    
  /home/User/Movies
    ├── Dir 01
    │   ├── subtitles.ass
    │   ├── Movie 01.mkv
    │   ├── chapters.xml
    │   └── tags.xml
    ├── Dir 02
    │   ├── Subtitles.ass
    │   ├── Movie 02.mkv
    │   ├── chapters.xml
    │   └── tags.xml
```

In the example above, \``/home/User/Movies`\` acts as the [*root* directory](#root-directory), this root directory contains two [source directories](#source-directory) inside it; namely, \``Dir 01`\` and \``Dir 02`\`.

And each of these source directories further contains a media file (`Movie XX.mkv`), a subtitle file, and accompanying chapters and tags.

## Installation

`auto-sub` is a Go program, it can be installed as a simple binary/executable file.

- [Download](https://github.com/demon-rem/auto-sub/releases) the relevant binary for your system.
- Extract the `auto-sub` or `auto-sub.exe` file from the archive
- Run `auto-sub -v` to test

Check out the [documentation](#documentation) for more info on how to use auto-sub.

Windows users can register the path to the executable as an [environment variable](https://stackoverflow.com/a/64233155/) to run auto-sub from command prompt without having to change directories.

Linux/Mac users can move the binary to `/usr/local/bin` (or `/usr/bin` depending on your requirements) to achieve the same result.

### Compiling from source

Note: These instructions are to generate an executable from the source-code by yourself. If you want an easier solution, check out the [installation section](#installation) to download a pre-compiled executable.

Make sure you have [Go](https://golang.org/) installed. [Download Go](https://golang.org/dl/) for your system if required.

```bash
git clone https://github.com/demon-rem/auto-sub
cd ./auto-sub
go build

./auto-sub -v
```

This will leave you with a checked out version of `auto-sub` that you can 

## Setup

auto-sub uses FFmpeg in the backend to modify the media files. Make sure to have FFmpeg and FFprobe installed in your system in order to use auto-sub.

Get pre-complied binaries and installation instructions for FFmpeg and FFprobe [here](https://ffmpeg.org/download.html)

## Documentation

Start by testing out your setup.

```
auto-sub --test
```

If the auto-sub fails to fetch versions for FFmpeg or FFprobe, head over to the [setup](#setup) section to install FFmpeg and FFprobe, make sure you have FFmpeg as well as FFprobe installed in your system. If the issue persists, consider filing a [bug-report](https://github.com/demon-rem/auto-sub/issues)

#### Syntax

Auto-sub is simply a wrapper over FFmpeg, its syntax is like this;

```bash
auto-sub ["/path/to/root"] [flags]
```

Note: In the command above, the only part compulsary is the path to the root/source directory. The path to the root directory can be provided as an argument, **or** through the appropriate flag.

### Flags

Flags can help you fine-tune the workings of auto-sub to meet your requirements, such as ignoring a particular file, or ignoring any file that meets a regex pattern, and more.

This section lists out all the possible flags accepted, and their default values (if any).

Note that **all** of these flags are optional - as such, auto-sub can work perfectly fine even without them!


### Boolean Flags

|    Flag   	| Short-hand 	|                     Purpose                    	|
|:---------:	|------------	|:----------------------------------------------:	|
|   --log   	|    none    	|        Generate logs for the current run       	|
|   --test  	|    none    	|        Run test(s) to verify your setup        	|
| --version 	|     -v     	|      Display current version for auto-sub      	|
|   --help  	|     -h     	|            Display help for auto-sub           	|
|  --direct 	|    none    	| Treat the root directory as a source directory 	|

<br>

### Miscellaneous Flags
|    Flag    	| Short-hand 	| Expected Value  	| Purpose                                          	| Default Value                      	| Required 	|
|:----------:	|------------	|-----------------	|--------------------------------------------------	|------------------------------------	|:--------:	|
|   --root   	|    none    	|      String     	| Path to the root directory                       	|                none                	|    No    	|
| --language 	|     -l     	|      String     	| Language code to be used with subtitles (if any) 	|                none                	|    No    	|
| --subtitle 	|    none    	|      String     	| Custom title to be used for the subtitle files   	| Original name of the subtitle file 	|    No    	|
|  --ffmpeg  	|    none    	|      String     	| Path to FFmpeg binary/executable                 	|         Decided at runtime         	|    Yes   	|
|  --ffprobe 	|    none    	|      String     	| Path to FFprobe binary/executable                	|         Decided at runtime         	|    Yes   	|
|  --exclude 	|     -E     	| List of strings 	| List of file names to be ignored                 	|                none                	|    No    	|
| --rexclude 	|    none    	|      String     	| String containing regex pattern to ignore files  	|                none                	|    No    	|


## License

Distributed under the MIT License. See [LICENSE](./LICENSE) for details.

## Roadmap

The main aim for this project is to act as a wrapper over FFmpeg - allowing users to soft sub (even multiple) files at once, without having to trudge through pages of documentation to learn the basics of FFmpeg.

A large part of this functionality is already present in the program, nevertheless, this section attempts to list out features that *may* be added in the future. Note that none of these features are intended to break/modify the existing functionality of auto-sub, rather add to what already exists, and simplify where possible.

***A list of possible improvements;***
 - Silent mode
 - Interactive mode
 - Config file (no gurantees)
 - Force flag (overwrite existing files - if any)
 - Custom naming for output files

Have a suggestion/feature in your mind that ins't listed here? Feel free to [file an issue](https://github.com/demon-rem/auto-sub/issues) :)

[code-size]: https://img.shields.io/github/languages/code-size/demon-rem/auto-sub?style=for-the-badge
[language]: https://img.shields.io/github/languages/top/demon-rem/auto-sub?style=for-the-badge
[license]: https://img.shields.io/github/license/demon-rem/auto-sub?style=for-the-badge
[latest-release]: https://img.shields.io/github/v/release/demon-rem/auto-sub?style=for-the-badge
[release-date]: https://img.shields.io/github/release-date/demon-rem/auto-sub?style=for-the-badge
[issues-url]: https://img.shields.io/github/issues-raw/demon-rem/auto-sub?style=for-the-badge
