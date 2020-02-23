import schedule
import time
import requests

def job():
    global counter

    now = int(time.time())
    print("Running requests - timestamp: %s (%s)" %
        (now, time.strftime("%d-%m-%Y %H:%M:%S", time.gmtime(now))))

    r = requests.get('https://devt.de')
    res_time = r.elapsed
    print ("  https://devt.de -> ", res_time)

schedule.every(2).seconds.do(job)

while True:
    schedule.run_pending()
    time.sleep(1)
