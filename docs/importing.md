# Importing

## Sources

Initial support:

- CSV imports

Future support:

- PSD2/Open Banking APIs
- FinTS/HBCI

## Pipeline

source -> raw import -> normalization -> deduplication -> canonical transaction

## Importer Design

Each importer:

- parses source format
- maps fields
- produces normalized transactions

Importers do not perform categorization.
