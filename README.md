# BirdNET-Pi2Go

## Overview
BirdNET-Pi2Go is a data migration tool designed to facilitate the conversion of BirdNET-Pi database contents and audio files to the BirdNET-Go data model. This utility ensures seamless transition between the two models by providing functionalities for database conversion and audio file transfer.

## Features

- Database Conversion: Converts BirdNET-Pi SQLite database to BirdNET-Go format.
- Audio File Transfer: Supports copying or moving BirdNET-Pi audio files to BirdNET-Go directory structure.
- Flexible Operation Modes: Allows users to choose between copying or moving audio files based on their needs.
- Disk Space Check: Verifies adequate disk space is available before performing copy operations.
- Skip Audio Transfer: Option to skip audio file transfer and only perform database migration.

## Requirements

Go programming language environment for building the tool.
Access to the file system containing BirdNET-Pi and BirdNET-Go data.

## Usage

### Building

To build BirdNET-Pi2Go from source, clone the repository and use the Go build command:

```bash
git clone https://github.com/tphakala/birdnet-pi2go.git
cd birdnet-pi2go
go build
```

### Running

After building, you can run BirdNET-Pi2Go with various flags to customize the migration process:

```bash
./birdnet-pi2go -source-db <path_to_birdnet_pi_db> -target-db <path_to_birdnet_go_db> -source-dir <path_to_birdnet_pi_audio_files> -target-dir <path_to_birdnet_go_audio_files> -operation <copy|move> -skip-audio-transfer <true|false>
```

#### Flags

- source-db: Path to the BirdNET-Pi SQLite database.
- target-db: Path to the BirdNET-Go SQLite database.
- source-dir: Path to BirdNET-Pi BirdSongs directory
- target-dir: Path to BirdNET-Go clips directory
- operation: Operation to perform on audio files (copy or move).
- skip-audio-transfer: Skip transferring audio files and only perform database migration (true/false).

### Example

```bash
./birdnet-pi2go -source-db birds.db -target-db birdnet.db -source-dir ~birdnetpi/BirdSongs -target-dir clips -operation copy
```

## Disclaimer

This tool is provided 'AS IS', without warranty of any kind. Please ensure you have backed up your data before using this tool. The developers are not responsible for any loss of data.

## Contributing

Contributions to BirdNET-Pi2Go are welcome. Please feel free to fork the repository, make your changes, and submit a pull request.
