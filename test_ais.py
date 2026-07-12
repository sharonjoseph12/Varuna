import asyncio
import websockets
import json

async def main():
    async with websockets.connect('wss://stream.aisstream.io/v0/stream') as websocket:
        msg = {
            'APIKey': 'e33255b8c258a8b550fe039f87028b5a86959737',
            'BoundingBoxes': [[[-90, -180], [90, 180]]],
            'FilterMessageTypes': ['PositionReport']
        }
        await websocket.send(json.dumps(msg))
        print('Sent msg')
        try:
            for i in range(5):
                res = await asyncio.wait_for(websocket.recv(), timeout=5)
                print(res[:200])
        except asyncio.TimeoutError:
            print('Timeout!')

asyncio.run(main())
