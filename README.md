# lifevisor

![image](https://github.com/user-attachments/assets/4db0d42b-9161-4476-9991-e97b6b5988fa)

---

# **Lifevisor**

**Lifevisor** is a quantified self-data platform designed to empower users with insights into their screentime habits. Built on top of ActivityWatch, it aggregates, analyses, and visualises screentime data to help users make informed decisions about their productivity and digital habits.
 
---

## **Tech Stack**
- **ActivityWatch**: Screentime tracking and data generation.
- **PostgreSQL**: Reliable database for syncing and storing activity data every 5 minutes.
- **Metabase**: Powerful visualization tool for creating customizable dashboards and reports.

---

## **Getting Started**

### **Prerequisites**
- [Go](https://go.dev/doc/install): Ensure Go is installed on your system.
- PostgreSQL: A running PostgreSQL database instance.
- ActivityWatch: Install and configure ActivityWatch to track screentime.

---

### **Installation**

1. **Clone the Repository**:
   ```bash
   git clone https://github.com/azaurus1/lifevisor.git
   cd lifevisor
   ```

2. **Build the CLI Application**:
   ```bash
   cd app
   go build -o lifevisor main.go
   ```
   This will generate the `lifevisor` binary in the current directory.

3. **Move the Binary to Your Path**:
   For easier usage, move the `lifevisor` binary to a directory in your system’s PATH:
   ```bash
   mv lifevisor /usr/local/bin/
   ```

---

### **Initial Setup**

Before syncing data, initialize the platform with the `init` command:

```bash
lifevisor init pg <activitywatch-db-path> <postgres-connection-string> <batch-size>
```

- **`<activitywatch-db-path>`**: Path to your ActivityWatch SQLite database file, typically `~/.local/share/activitywatch/aw-server/peewee-sqlite.v2.db`.
- **`<postgres-connection-string>`**: Connection string for your PostgreSQL database (e.g., `postgres://username:password@host:port/dbname`).
- **`<batch-size>`**: Number of records to process in each batch during synchronization.

**Example**:
```bash
lifevisor init pg ~/.local/share/activitywatch/aw-server/peewee-sqlite.v2.db postgres://postgres:postgres@192.168.0.131:30136/lifevisor 10000
```

---

### **Set Up the Cronjob**

To keep your data up-to-date, configure a cronjob to run the `sync` command every 5 minutes:

1. Open your crontab editor:
   ```bash
   crontab -e
   ```

2. Add the following entry:
   ```bash
   */5 * * * * /usr/local/bin/lifevisor sync pg ~/.local/share/activitywatch/aw-server/peewee-sqlite.v2.db postgres://postgres:postgres@192.168.0.131:30136/lifevisor 300
   ```

- **`sync` Command**: Updates the PostgreSQL database with the latest data from ActivityWatch.
- **`<activitywatch-db-path>`**: Same as above.
- **`<postgres-connection-string>`**: Same as above.
- **`300`**: Sync records within the last 300 seconds (5 minutes).

**Note**: Replace `/usr/local/bin/lifevisor` with the actual path to your `lifevisor` binary if it differs.

---

### **Verify Setup**

1. Run the `sync` command manually to ensure it works:
   ```bash
   lifevisor sync pg ~/.local/share/activitywatch/aw-server/peewee-sqlite.v2.db postgres://postgres:postgres@192.168.0.131:30136/lifevisor 300
   ```

2. Check your PostgreSQL database to confirm that screentime data is being updated.

3. Verify that the cronjob is running as expected by checking your system logs:
   ```bash
   grep CRON /var/log/syslog
   ```
