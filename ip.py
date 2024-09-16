from stem import Signal
from stem.control import Controller
import time
import requests

def get_tor_ip_via_privoxy():
    """
    Gets the external IP address being used by Tor through Privoxy.
    """
    proxies = {
        'http': 'http://localhost:8118',  # Privoxy's default port
        'https': 'http://localhost:8118',
    }
    try:
        # Making a request to an external IP check service via the Privoxy proxy
        response = requests.get('http://httpbin.org/ip', proxies=proxies)
        return response.json()['origin']
    except Exception as e:
        return f"Error fetching IP via Privoxy: {e}"

# Connect to Tor control port
with Controller.from_port(port=9051) as controller:
    controller.authenticate()  # Automatically handles cookie authentication

    while True:
        # Change IP (new Tor circuit)
        controller.signal(Signal.NEWNYM)

        # Give Tor a moment to switch circuits
        time.sleep(3)

        # Get the current Tor IP via Privoxy
        new_ip = get_tor_ip_via_privoxy()
        print(f"Tor IP changed to: {new_ip}")

        # Wait a seconds before changing IP again
        time.sleep(7)
