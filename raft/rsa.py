from cryptography.hazmat.primitives.asymmetric import rsa
from cryptography.hazmat.primitives.serialization import Encoding
from cryptography.hazmat.primitives.serialization import PrivateFormat
# from cryptography.hazmat.primitives.serialization import BestAvailableEncryption
from cryptography.hazmat.primitives.serialization import PublicFormat
from cryptography.hazmat.primitives import serialization

private_key_pass = b"0123456789"

ENCODING = Encoding.PEM
PRIVATE_FORMAT = PrivateFormat.PKCS8
PUBLIC_FORMAT = PublicFormat.SubjectPublicKeyInfo
ENCRYPTION = serialization.NoEncryption()

private_key = rsa.generate_private_key(
    public_exponent=65537,
    key_size=2048,
)

pem_private_key = private_key.private_bytes(
    ENCODING,
    PRIVATE_FORMAT,
    ENCRYPTION
    )

print(f"Private Key: {pem_private_key}")

pem_public_key = private_key.public_key().public_bytes(
  encoding=ENCODING,
  format=PUBLIC_FORMAT
)

print()
print(f"Public Key: {pem_public_key}")




from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.primitives.asymmetric import padding
message = bytes(str(pem_public_key), "utf-8")
signature = private_key.sign(
    message,
    padding.PSS(
        mgf=padding.MGF1(hashes.SHA256()),
        salt_length=padding.PSS.MAX_LENGTH
    ),
    hashes.SHA256()
)

print()
print(f"Signature: {signature}")

public_key = private_key.public_key()
matches: bool
try:
    public_key.verify(
        signature,
        message,
        padding.PSS(
            mgf=padding.MGF1(hashes.SHA256()),
            salt_length=padding.PSS.MAX_LENGTH
        ),
        hashes.SHA256()
    )
    matches = True
except:
    matches = False


print()
print(f"Verification: {matches}")