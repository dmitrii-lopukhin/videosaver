import os
import sys

# Ensure `import app.*` resolves when pytest runs from the insta-resolver dir.
sys.path.insert(0, os.path.dirname(__file__))
