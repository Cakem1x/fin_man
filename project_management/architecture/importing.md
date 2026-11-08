# Importing

## Sources

Supported:

- CSV imports (`fin import-csv -c <config> <file>`)

Future support:

- PSD2/Open Banking APIs
- FinTS/HBCI

## Pipeline

source -> parsing -> normalization -> deduplication -> insert (source file archived)

1. **Parsing**: A `genericcsv` config maps CSV columns to transaction fields. `-c` takes a builtin config name (e.g. `dkb_v1`) or a path to a JSON config file.
2. **Normalization**: Cleans whitespace and title-cases the payee.
3. **Deduplication**: The transaction ID is a SHA-256 hash of date, lowercased payee, amount, currency, and the memo reduced to lowercase letters and digits. A transaction whose ID already exists is a duplicate and is not inserted. This makes overlapping imports safe.
4. **Archiving**: When at least one transaction is new, the source file is copied to `store/archives/csv_<config>/<timestamp>_<file>`. Each transaction stores this path in `archive_file_path`.

## Importer Design

Each importer:

- parses the source format
- maps fields
- produces normalized transactions

Importers do not write to the database and do not perform categorization.
