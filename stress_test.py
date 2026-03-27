import csv
from locust import HttpUser, task, between, constant
from queue import Queue, Empty

def load_tokens():
    token_queue = Queue()
    with open("test_tokens.csv", "r") as f:
        reader = csv.DictReader(f)
        for row in reader:
            token_queue.put(row["token"])
    return token_queue

# 全局 Token 池
TOKEN_POOL = load_tokens()

class GTicketUser(HttpUser):
    # 修改：将等待时间设为 0，全力冲击服务器
    wait_time = constant(0) 
    
    def on_start(self):
        try:
            # 每个虚拟用户从池子里拿一个唯一的 Token
            self.token = TOKEN_POOL.get_nowait()
        except Empty:
            self.token = None
            print("警告：Token 已耗尽，停止发压")

    @task
    def create_order(self):
        if not self.token:
            self.user.stop() # 没 Token 了就停止这个虚拟用户
            return

        headers = {"Authorization": self.token}
        payload = {"activityId": 1, "need": 1}

        with self.client.post("/api/v1/orders", json=payload, headers=headers, catch_response=True) as response:
            if response.status_code == 200:
                res = response.json()
                if res.get("code") == 200:
                    response.success()
                else:
                    response.failure(f"业务失败: {res.get('msg')}")
            else:
                # 400 也会在这里被标记为 Failure
                response.failure(f"HTTP Error: {response.status_code}")