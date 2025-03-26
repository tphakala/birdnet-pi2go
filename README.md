# 🐦 BirdNET-Pi2Go

## 🌟 Overview
BirdNET-Pi2Go is a data migration tool designed to facilitate the conversion of BirdNET-Pi database contents and audio files to the BirdNET-Go data model. This utility ensures seamless transition between the two systems while maintaining your valuable bird detection data.

## ✨ Features

- 🔄 **Database Conversion**: Migrates BirdNET-Pi SQLite database to BirdNET-Go format
- 📁 **Audio File Transfer**: Supports copying or moving audio recordings to BirdNET-Go directory structure
- 🔀 **Flexible Operations**: Choose between copying files (preserving originals) or moving files (saving space)
- 💾 **Disk Space Verification**: Automatically checks for sufficient storage before starting transfers
- ⏩ **Skip Audio Option**: Option to migrate database only, without transferring audio files
- 🔄 **Merge Support**: Ability to merge existing BirdNET-Go database with migrated data

## 📋 Requirements

- 🖥️ Go 1.21 or newer for building from source
- 📂 Access to BirdNET-Pi and BirdNET-Go file systems

## 🚀 Getting Started

### 🔨 Building

To build BirdNET-Pi2Go from source:

```bash
git clone https://github.com/tphakala/birdnet-pi2go.git
cd birdnet-pi2go
go build
```

### 📝 Usage Guide

After building, run BirdNET-Pi2Go with various flags to customize your migration:

```bash
./birdnet-pi2go -source-db <path_to_birdnet_pi_db> -target-db <path_to_birdnet_go_db> -source-dir <path_to_birdnet_pi_audio_files> -target-dir <path_to_birdnet_go_audio_files> -operation <copy|move> -skip-audio-transfer <true|false>
```

#### 🎛️ Command Options

| Flag | Description | Default |
|------|-------------|---------|
| `-source-db` | Path to BirdNET-Pi SQLite database | `birds.db` |
| `-target-db` | Path to BirdNET-Go SQLite database (will be created) | `birdnet.db` |
| `-source-dir` | Path to BirdNET-Pi BirdSongs directory | (required for file transfer) |
| `-target-dir` | Path to BirdNET-Go clips directory | `clips` |
| `-operation` | File transfer mode: `copy` or `move` | `copy` |
| `-skip-audio-transfer` | Skip audio file transfer (`true` or `false`) | `false` |

> ⚠️ **Note**: Target database should not exist - it will be created during migration.

### 🧪 Examples

#### Basic migration with file copying:
```bash
./birdnet-pi2go -source-db birds.db -target-db birdnet.db -source-dir ~/birdnetpi/BirdSongs -target-dir clips -operation copy
```

#### Migrate database only (no audio files):
```bash
./birdnet-pi2go -source-db birds.db -target-db birdnet.db -skip-audio-transfer true
```

#### Move files instead of copying (saves disk space):
```bash
./birdnet-pi2go -source-db birds.db -target-db birdnet.db -source-dir ~/birdnetpi/BirdSongs -target-dir clips -operation move
```

#### Merge existing databases:
```bash
./birdnet-pi2go -source-db birds.db -target-db birdnet.db -operation merge
```

## 📊 Data Handling

BirdNET-Pi2Go carefully preserves your detection data while converting between formats:
- 🔍 Detection records are mapped to BirdNET-Go's Note structure
- 🔊 Audio filenames are standardized according to BirdNET-Go conventions
- 🗂️ File organization follows BirdNET-Go's year/month directory structure

## ⚠️ Disclaimer

This tool is provided 'AS IS', without warranty of any kind. **Please ensure you have backed up your data before using this tool**. The developers are not responsible for any loss of data.

## 📜 License

MIT
