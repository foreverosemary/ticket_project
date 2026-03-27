import requests
import csv

BASE_URL = "http://127.0.0.1:8080/api/v1"
USER_COUNT = 2000  # 准备 100 个压测账号

def prepare():
    users_data = []
    print(f"正在准备 {USER_COUNT} 个用户...")

    for i in range(1, USER_COUNT + 1):
        username = f"testuser_{i}"
        password = "password123"

        # 1. 注册 (如果已存在会报错，这里忽略错误)
        requests.post(f"{BASE_URL}/users", data={"username": username, "password": password})

        # 2. 登录获取 Token
        resp = requests.post(f"{BASE_URL}/users/login", data={"username": username, "password": password})
        
        if resp.status_code == 200:
            data = resp.json()
            token = data["data"]["token"]
            user_id = data["data"]["userId"]
            users_data.append([user_id, token])
            print(f"用户 {username} 准备就绪")

    # 3. 写入 CSV 文件
    with open("test_tokens.csv", "w", newline="") as f:
        writer = csv.writer(f)
        writer.writerow(["user_id", "token"]) # 表头
        writer.writerows(users_data)
    
    print(f"成功导出 {len(users_data)} 个用户 Token 到 test_tokens.csv")

if __name__ == "__main__":
    prepare()