/*
Package rate-limiting implements a middleware to rate limit IBC transfers
between different chains to prevent excessive token flow in either direction.
This module monitors and enforces configurable rate limits on token transfers
across IBC channels to protect chains from economic attacks or unintended
token drainage.
*/
package ratelimiting
