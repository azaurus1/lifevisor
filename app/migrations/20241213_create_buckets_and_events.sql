-- +migrate Up
-- Create the Bucket table
CREATE TABLE bucketmodel (
    key SERIAL PRIMARY KEY, -- Auto-incrementing primary key
    id TEXT NOT NULL UNIQUE, -- Unique string identifier
    created TIMESTAMP NOT NULL DEFAULT NOW (), -- Creation timestamp with default value
    name TEXT NOT NULL, -- Bucket name
    type TEXT NOT NULL, -- Bucket type
    client TEXT NOT NULL, -- Client associated with the bucket
    hostname TEXT NOT NULL -- Hostname for the bucket
);

-- Create the Event table
CREATE TABLE eventmodel (
    id SERIAL PRIMARY KEY, -- Auto-incrementing primary key
    bucket_id INT NOT NULL, -- Foreign key referencing Bucket.Key
    timestamp TIMESTAMP NOT NULL, -- Event timestamp
    duration FLOAT NOT NULL, -- Duration of the event
    datastr JSON NOT NULL, -- JSON data associated with the event
    FOREIGN KEY (bucket_id) REFERENCES bucketmodel (key) ON DELETE CASCADE -- Cascade delete
);

-- +migrate Down
-- Drop the Event table
DROP TABLE IF EXISTS Event;

-- Drop the Bucket table
DROP TABLE IF EXISTS Bucket;
