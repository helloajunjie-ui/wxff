import hashlib

media_id = "14799503133509825267"
key = "198511700483708433"
token = "qe9uKlQ2RT8F1FD44V7hbiaGw3FzfJpsGZ832cBdvEuDWHesvC84WjGx5BZc5vDM0qbQxOgicvjoQ"

print("md5(media_id):", hashlib.md5(media_id.encode()).hexdigest())
print("md5(key):", hashlib.md5(key.encode()).hexdigest())
print("md5(token):", hashlib.md5(token.encode()).hexdigest())

# URL 里的 hash
url_hash = "xnaCgLkwO2WBPBmYVwrs2yGzyP1gr96QKVgoPNSEpWtpcYYibQiaEV9WdKjdvCyxXsfJ8NyDlWY7k"
print("md5(url_hash):", hashlib.md5(url_hash.encode()).hexdigest())
