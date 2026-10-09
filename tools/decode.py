import base64
s = "GhQKEnd4YWFkYmFiOWQxM2VkZmYyMA=="
decoded = base64.b64decode(s)
print("hex:", decoded.hex())
print("raw:", decoded)
