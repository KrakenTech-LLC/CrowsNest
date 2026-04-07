<div align="center">
    <img src=.img/crowsnest.png  style="width: 500px; height: auto" alt="CrowsNest Logo" title="CrowsNest Logo">
</div>

### A CLI tool for seamless interaction with the Dehashed and Hunter.io APIs.

---

## 🌟 Features
- **Output Format Control**: JSON, YAML, XML, and TEXT support.
- **Regex & Wildcard Matching**: Flexible query options.
- **Local Database Storage**: Default or custom paths.
- **Database Querying**: Raw SQL and filtered queries.
- **Enhanced Logging**: Easy log parsing and rotation.
- **Error Handling**: Intelligent API error management.
- **WhoIs Lookups**: Domain, IP, MX, NS, and more.
- **Subdomain Scanning**: Identify subdomains.
- **Robust Logging**: Detailed logs for debugging.
- **API Key Management**: Securely store and manage API keys.
- **Formatted Output**: Easy to read and understand.
- **Intuitive Database Querying**: Query for specific information.
- **Person and Company Enrichment**: Retrieve detailed information about people and companies.
- **Email Verification**: Verify the existence and quality of email addresses.

---

## 🔰 Getting Started

To begin, clone the repository
``` bash-session
git clone https://github.com/Kraken-OffSec/CrowsNest.git
cd crowsnest
go build crowsnest.go
```

---

## 🗄️ Database Configuration

CrowsNest supports two database storage options:

1. **Default Path** (default): Stores the database at `~/.local/share/crowsnest/db/crowsnest.sqlite`
2. **Local Path**: Stores the database in the current directory as `./crowsnest.sqlite`

The **Local Path** option allows for separate databases for different projects or engagements.

To configure the database location:

```bash
# Use local database in current directory
./crowsnest set-local-db true

# Use default database path
./crowsnest set-local-db false
```

---

## 🌐 Dehashed

### Initial Setup

CrowsNest requires an API key from Dehashed. Set it up with:
![Alt text](.img/set-dehashed.png "Set Dehashed Key")
```bash
ar1ste1a@kali:~$ crowsnest set dehashed <redacted>
```

### Simple Query
CrowsNest can be used simply for example to query for credentials matching a given email domain.
![Alt text](.img/simple_query.png "Simple Query")
``` go
# Provide credentials for domains matching target.com
crowsnest dehashed -D target.com
```

### Simple Credentials Query
CrowsNest can also be used to return only credentials for a given query.
![Alt text](.img/simple_creds_query.png "Creds Only Query")
``` go
# Provide credentials for emails matching @target.com
crowsnest dehashed -D @target.com -C
```

### Wildcard Query
CrowsNest is capable of handling wildcard queries.  
A wildcard query cannot begin with a wildcard.  
This is a limitation of the Dehashed API.
An asterisk can be used to denote multiple characters, and a question mark can be used to denote a single character.
<br>
![Alt text](.img/wildcard_query.png "Wildcard Query")
``` go
# Provide credentials for emails matching @target.com and @target2.com
crowsnest dehashed -E @target?.com -C -W
```

### Email Query
Dehashed has dictated that emails should be searched in the following format:
`email:target.name&domain:target.com`.
As such, to query an email, please use the following format (note, wildcard is not required but can be useful):
<br>
*see photo above in Wildcard Query*
``` go
# Provide credentials for emails matching target.*@target.com
crowsnest dehashed -W -E 'target*' -D target.com
```
You may also query the domain and find emails as well
``` go
# Provide credentials for emails matching target.com
crowsnest dehashed -D target.com -C
```

### Combining Queries
CrowsNest is capable of combining queries.  
This is useful for when you want to query for credentials matching a given email or domain, but only for a specific username.
![Alt text](.img/combining_queries.png "Combined Query")
``` go
# Provide credentials for emails matching @target.com and username containing 'admin'
crowsnest dehashed -D target.com -U admin
```

### Regex Query
CrowsNest is capable of handling regex queries.  
Simply denote regex queries with the `-R` flag.
Place all regex queries in quotes with the corresponding query flag in single quotes.
<br>
!!!! *Currently, the Regex Operators appear to be broken. I am waiting on a response from Dehashed* !!!!
``` go
# Return matches for emails matching this given regex query
crowsnest dehashed -R -E 'joh?n(ath[oa]n)' -D hotmail.com'
```

### Output Text (default JSON)
CrowsNest is capable of handling output formats.  
The default output format is JSON.  
To change the output format, use the `-f` flag.  
CrowsNest currently supports JSON, YAML, XML, TEXT, and GREP output formats.
``` go
# Return matches for usernames exactly matching "admin" and write to text file 'admins_file.txt'
crowsnest dehashed -U admin -o admins_file -f txt

# Return one key=value record per line in a greppable file 'admins_file.grep'
crowsnest dehashed -U admin -o admins_file -f grep
```

### Data Wells
DeHashed data wells are free to query and do not require a paid API account.
``` go
# List the first page of data wells and write 'data_wells.json'
crowsnest dehashed data-wells

# Sort by record count and write one key=value record per line
crowsnest dehashed data-wells --sort records-DESC --count 50 -f grep -o data_wells
```

---

## 🌐 WhoIs Lookups
CrowsNest supports WHOIS lookups, history searches, reverse WHOIS searches, IP lookups, MX lookups, NS lookups, and subdomain scans.
The WhoIs Lookups require a separate API Credit from the Dehashed API.

### Domain Lookup
CrowsNest can perform a domain lookup for a given domain.
This provides a tree view of the domain's WHOIS information.
![Alt text](.img/whois_domain.png "WhoIs Tree View")
```bash
# Perform a WHOIS lookup for example.com
crowsnest whois -d example.com
```

### History Lookup
History Lookups require 25 credits. 
This is a Dehashed API limitation.
The history lookup is immediately written to file and not displayed in the terminal or stored in the database.
```bash
# Perform a WHOIS history search for example.com
crowsnest whois -d example.com -H
```

### Subdomain Scan
CrowsNest can perform a subdomain scan for a given domain.  
This provides a list of all subdomains that match the given query.
![Alt text](.img/whois_subdomain.png "WhoIs Tree View")
```bash
# Perform a WHOIS subdomain scan for google.com
crowsnest whois -d google.com -s
```

### Reverse WHOIS Lookup
CrowsNest can perform a reverse WHOIS lookup for given criteria.  
This provides a list of all domains that match the given query.  
The reverse WHOIS lookup is immediately written to file and not stored in the database.
![Alt text](.img/whois_reverse.png "WhoIs Tree View")
```bash
# Perform a reverse WHOIS lookup for example.com
crowsnest whois -I example.com
```

### IP Lookup
CrowsNest can perform a reverse IP lookup for a given IP address.  
This provides a list of all domains that match the given query.
![Alt text](.img/whois_ip.png "WhoIs View")
```bash
# Perform a reverse IP lookup for 8.8.8.8
crowsnest whois -i 8.8.8.8
```

### MX Lookup
CrowsNest can perform an MX lookup for a given MX hostname.  
This provides a list of all domains that match the given query.
![Alt text](.img/whois_mx.png "WhoIs Tree View")
```bash
# Perform a reverse MX lookup for google.com
crowsnest whois -m stmp.google.com
```
### NS Lookup
CrowsNest can perform an NS lookup for a given NS hostname.  
This provides a list of all domains that match the given query.
The picture below also includes the --debug global flag.
![Alt text](.img/whois_ns.png "WhoIs Tree View")
```bash
# Perform a reverse NS lookup for google.com
crowsnest whois -n google.com
```


---

## 🌐 Hunter.io
CrowsNest supports Hunter.io lookups.  
Hunter.io lookups require a separate API Key from the Dehashed API.
This can be set using the `set hunter` command.
![Alt text](.img/set-hunter.png "Set Dehashed Key")
```bash
# Set the Hunter.io API key
crowsnest set hunter <redacted>
```

### Domain Search
CrowsNest can perform a domain search for a given domain.  
This provides information about  company including a description, social media information and any technologies in use.<br>
![Alt text](.img/hunter_domain.png "Hunter.io Domain Search")
```bash
# Perform a Hunter.io domain search for example.com
crowsnest hunter -d example.com -D
```

### Email Verification
CrowsNest can perform an email verification search for a given email.  
This provides a verification and score of a given email address.
![Alt text](.img/hunter_emailverification.png "Hunter.io Email Verification")
```bash
# Perform a Hunter.io email verification search for example@target.com
crowsnest hunter -e example@target.com -V
```

### Company Enrichment
CrowsNest can perform a company enrichment search for a given domain.  
This provides information about a company given its domain.
![Alt text](.img/hunter_company.png "Hunter.io Company Enrichment")
```bash
# Perform a Hunter.io company enrichment search for example.com
crowsnest hunter -d example.com -C
```

### Email Finder
CrowsNest can perform an email finder search for a given domain, first name, and last name.  
This provides information about a user including a confidence score, and any social media accounts linked to a first name, last name and email.
![Alt text](.img/hunter_emailfind.png "Hunter.io Email Finder")
```bash
# Perform a Hunter.io email finder search for example.com
crowsnest hunter -d example.com -F John -L Doe -E
```

### Person Enrichment
CrowsNest can perform a person enrichment search for a given email.  
This provides information about a user given an email address..
![Alt text](.img/hunter_person.png "Hunter.io Person Enrichment")
```bash
# Perform a Hunter.io person enrichment search for example@target.com
crowsnest hunter -e example@target.com -P
```

### Combined Enrichment
CrowsNest can perform a combined enrichment search for a given email.  
This is a combination of the company and person enrichments given an email address.
![Alt text](.img/combined_enrichment_1.png "Hunter.io Combined Enrichment")
![Alt text](.img/combined_enrichment_2.png "Hunter.io Combined Enrichment")
```bash
# Perform a Hunter.io combined enrichment search for example@target.com
crowsnest hunter -e example@target.com -B
```
## Debugging
CrowsNest supports debugging.  This can be enabled using the `--debug` flag in the root command.
![Alt text](.img/crowsnest_debugging_global.png "Debugging")
```bash
# Perform a Hunter.io combined enrichment search for example@target.com with debugging enabled
crowsnest --debug hunter -e example@target.com -B
```
---
## 📊 Database Querying
CrowsNest stores query results in a local database.  
This database can be queried for previous results.
This is useful for when you want to query for specific information.
This database also includes WhoIs Information and Subdomain Scan results, but does **not** include historical lookups.

## Simple Query
#### It's possible to query the database using shorthand and without knowing any SQL at all.
#### The following queries the results table where username is not null, only showing the username, email and password columns.
![Alt text](.img/query_simple.png "Simple Query")
#### You may also add in a simple query using the `-q` flag. The following displays a 'LIKE' clause on the email column.
#### Note the '%\<clause\>%' is still required.
![Alt text](.img/query_where.png "Simple Query")
  
```bash
# Query the database for all results containing the word 'admin' in the username
crowsnest query -t results -q "username LIKE '%admin%'"
```


## Raw SQL Queries
![Alt text](.img/query_raw.png "Raw Query")

CrowsNest also supports raw SQL queries.  This is useful for when you want to query for specific information.
```bash
# Query the database for all results containing the word 'admin' in the username
crowsnest query -r "SELECT * FROM results WHERE username LIKE '%admin%'"
```

## Query Options
CrowsNest supports a number of query options.  These options can be used to filter the results of a query.
```bash
# Query the database for all results containing the word 'admin' in the username
crowsnest query -t results -q "username LIKE '%admin%'" -n username,email,password
```

## Listing Tables and Columns
CrowsNest supports listing all available tables and columns.  
This is useful for when you want to query for specific information.
![Alt text](.img/query_alltables.png "List All Tables")
```bash
# List all available tables and columns
crowsnest query -a
```

The current tables available for query are:
- results
  - Results from a dehashed query
- creds
  - Credentials parsed from dehashed results
- whois
  - Results from a whois record lookup
- subdomains
  - Subdomains discovered in a whois subdomain scan 
- runs
  - Previous query runs to the dehashed API
- lookup 
  - Results of any Whois NS, MX, or IP lookup
- hunter_domain
  - Results from a hunter.io domain search
- hunter_email
  - Results extracted from a domain saerch and email finder.

---

# Exporting Results
CrowsNest supports exporting results to a file.  
This is useful for when you want to requery for specific information without touching the Dehashed API.
The export subcommand supports all the same options as the query subcommand.
The export subcommand also supports file naming and output format control.
![Alt text](.img/export_raw.png "Export Results")
```bash
# Export all results containing the word 'admin' in the username to a text file
crowsnest export -t results -q "username LIKE '%admin%'" -o admins_file -f txt
```

## 🐛 Debugging

CrowsNest uses the `zap` logging library for logging.  The logs are stored in `~/.local/share/crowsnest/logs`.
The logs can be easily queried from the crowsnest CLI.

### Filtering by Date
#### CrowsNest utilizes 'easy time' to determine the appropriate time for a given query.
![Alt text](.img/easy_time_parsing.png "Easy Time")
#### You may also used dates mixed with easy time to perform queries.
![Alt text](.img/mixed_time_query.png "Mixed Time")
#### The following formats are supported:
- `last 24 hours`
- `last 2 days`
- `30 minutes ago`
- `45 seconds ago`
- `1 week ago`
- `05-01-2025`
- `05/01/2025`
- `05/01/25`
- `05-01-25`
- `May 01, 2025`

```bash
# Show the last 10 logs
crowsnest logs -l 10

# Show logs from the last 24 hours
crowsnest logs -s "last 24 hours"

# Show logs from the last 24 hours with a severity of error or fatal
crowsnest logs -s "05-01-2025" -v error,fatal
```

## 🤝 Contributing
Contributions are welcome! Submit a pull request to help improve CrowsNest.

## [Buy Me A Coffee](https://buymeacoffee.com/ehosinskiz)

<div align="center">
    <img src=.img/kraken_k.png style="width: 350px; height: auto" alt="Ar1ste1a" title="Ar1ste1a Offensive Security">
</div>

## **Release The Kraken**
