import asyncio

from .main import main

try:
    asyncio.run(main())
except KeyboardInterrupt:
    pass
