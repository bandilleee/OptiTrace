# Supabase database trust anchor

`supabase-root-2021.crt` is the public Supabase Root 2021 CA downloaded from the project's Database Settings **Download certificate** link:

https://supabase-downloads.s3-ap-southeast-1.amazonaws.com/prod/ssl/prod-ca-2021.crt

Verified 2026-10-25. SHA256 certificate fingerprint: `80:70:25:AD:50:D4:ED:21:9D:2C:9C:7D:29:9C:00:4F:82:4E:B0:0C:F7:F6:5A:FE:F6:07:D0:7B:72:E6:CA:FA`. Valid until 2031-04-26. This contains no private key or account credential.

The portfolio image copies it to `/app/certs/supabase-root-2021.crt`. Its database URL must use `sslmode=verify-full&sslrootcert=/app/certs/supabase-root-2021.crt`. A system trust store alone did not validate the actual shared-pooler certificate. Do not weaken TLS; review any replacement CA against the provider's official settings before updating this file.
