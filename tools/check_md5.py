import hashlib
wxid = "wxid_5wi83mtrxlpt21"
print(f"wxid: {wxid}")
print(f"md5: {hashlib.md5(wxid.encode()).hexdigest()}")
