# Transaction Enrichment

## Overview

Transaction Enrichment (specifically the **Receipt Matcher** feature) is designed to solve the "Opaque Transaction Problem". Often, bank imports contain aggregated transactions with vague payee information or memos (e.g., an Amazon charge for $45.12 that gives no indication of the actual items purchased).

The Receipt Matcher allows `fin_man` to ingest external itemized data (like merchant order history) and use it to split the opaque bank transaction into distinct, line-item child transactions.

## Architecture & Auditability

To maintain a strict audit trail and comply with the principle that "raw imports are immutable", `fin_man` uses a parent-child database architecture for splits.

- **`transactions` table (`parent_id`)**: A foreign key pointing back to another transaction.
- **Immutability**: When a transaction is split, the original transaction is not deleted or overwritten. Instead, the new line items are inserted into the database with their `parent_id` set to the ID of the original bank transaction.
- **Visibility**: Queries filtering for active transactions (like `GetUnreviewedTransactions` or `GetAllTransactions`) automatically hide any transaction that acts as a parent (`WHERE NOT EXISTS (SELECT 1 FROM transactions child WHERE child.parent_id = t.id)`). This cleanly encapsulates the "hidden" state inside relational logic without requiring an explicit boolean toggle.
- **Handling Mismatches**: If the sum of the exact line-item amounts does not equal the bank transaction amount (e.g., due to shipping or taxes not captured in the item lines), the system automatically creates a "Remainder" child transaction to ensure the budget balances perfectly.

## TUI Workflow

Enrichment is natively integrated into the `fin` unified CLI's Review mode.
While focused on a transaction, pressing `[m]` opens the **Receipt Matcher**. A file picker UI appears, allowing the user to select the appropriate data export file. The system then parses the file, attempts a match, and immediately applies the split if successful.

## Supported Adapters

### Amazon Privacy CSV (`AmazonPrivacyCSVMatcher`)

The first implemented enrichment adapter is for Amazon purchases.

**Data Source**: The official Amazon Privacy Central data request CSV. This is preferred over third-party scrapers because the official CSV contains exact, per-item historical pricing data that matches the final billed amounts.

**Matching Logic**:
1. The adapter reads the CSV and dynamically locates the "Order ID", "Product Name / Title", and "Unit Price / Total" columns.
2. It strips currency symbols and converts the price into cents.
3. During the `Match` phase, it scans the bank transaction's memo for a standard Amazon Order ID pattern (which Amazon includes on bank statements).
4. If a matching Order ID is found in the CSV map, the adapter returns the itemized list, and the UI splits the transaction accordingly.
