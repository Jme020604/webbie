import os
import requests
import datetime
from concurrent.futures import ThreadPoolExecutor
from tqdm import tqdm

def check_url(good_url, show_403, status):
    response = requests.get(url=good_url)
    if response.status_code != 404:
        if show_403 == 'yes':
            now = datetime.datetime.now()
            print(f'[{now.time().strftime("%H:%M:%S")}][{response.status_code}]: {good_url} [{status}]')
        elif show_403 == 'no':
            if response.status_code != 403:
                now = datetime.datetime.now()
                print(f'[{now.time().strftime("%H:%M:%S")}][{response.status_code}]: {good_url} [{status}]')

def main(user_url, option, show_403):
    dirs = []
    if option == 'small':
        dir_list = '~/Documents/projectSec/webbie/dirLists/directory-list-2.3-small.txt'
    elif option == 'medium':
        dir_list = '~/Documents/projectSec/webbie/dirLists/directory-list-2.3-medium.txt'
    elif option == 'big':
        dir_list = '~/Documents/projectSec/webbie/dirLists/directory-list-2.3-big.txt'
    else:
        return dir_list
    
    dir_list = os.path.expanduser(dir_list)

    with open(dir_list, 'r') as data:
        for line in data:
            if not line.startswith('#'):
                line = line.replace("\n", "")
                if line != '':
                    dirs.append(line)

    # Create a ThreadPoolExecutor to process URLs in parallel
    total_urls = len(dirs)
    processed_urls = 0

    with ThreadPoolExecutor(max_workers=5) as executor, tqdm(total=total_urls, desc="Processing URLs") as pbar:
        for directory in dirs:
            good_url = user_url + '/' + directory
            status = f'{processed_urls + 1}/{total_urls}'
            executor.submit(check_url, good_url, show_403, status)
            processed_urls += 1
            pbar.update(1)

if __name__ == '__main__':
    user_url = input('site url: ')
    user = input('scan size: ')
    main(user_url, user, show_403='no')
