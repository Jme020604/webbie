import requests
import re

user_url = 'https://www.digitalocean.com/'

response = requests.head(url=user_url)

print(response)

# # Check if the 'Server' header exists in the response headers
# if 'Server' in response.headers:
#     server_header = response.headers['Server']
    
#     # Use re.search on the server_header to find the server name
#     server_match = re.search(r'([^\r\n]+)', server_header)

#     if server_match:
#         server_name = server_match.group(1)
#         print("Server:", server_name)
#     else:
#         print("Server name not found in the 'Server' header.")
# else:
#     print("Server header not found in the response headers.")

