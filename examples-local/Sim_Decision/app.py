#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
SimDecision - 决策软件
调用 SimRoutePlan、SimNaviControl 和 SimSonar 三个服务，完成完整的决策流程
"""

from flask import Flask, render_template, jsonify, request
import requests
import json
import time
from datetime import datetime
import os

app = Flask(__name__, template_folder='templates', static_folder='static')

# 服务地址配置（可通过环境变量覆盖）
CONTROLLER_URL = os.getenv('CONTROLLER_URL', 'http://localhost:8080')
SIM_ROUTE_PLAN_URL = os.getenv('SIM_ROUTE_PLAN_URL', 'http://localhost:3100')
SIM_NAVI_CONTROL_URL = os.getenv('SIM_NAVI_CONTROL_URL', 'http://localhost:3200')
SIM_SONAR_URL = os.getenv('SIM_SONAR_URL', 'http://localhost:3300')

# 服务名称映射（使用服务发现中的实际服务名）
SERVICE_NAMES = {
    'route_plan': 'planRoute1',  # SimRoutePlan 注册的服务名（使用 planRoute1）
    'navi_control': 'controlUSV',  # SimNaviControl 注册的服务名
    'sonar': 'detectTarget',  # SimSonar 注册的服务名
    'target_recognize': 'recognizeTarget',  # SimTargetRecognize 注册的服务名
    'target_hit': 'hitTarget'  # SimTargetHit 注册的服务名
}

# 请求超时设置（秒）
REQUEST_TIMEOUT = 60  # SimSonar 需要 3-5 秒，设置 60 秒超时

class ServiceCaller:
    """服务调用器"""
    
    def __init__(self):
        self.results = {
            'route_plan': None,
            'navi_control': None,
            'sonar': None,
            'target_recognize': None,
            'target_hit': None
        }
        self.errors = {}
        self.start_time = None
        self.end_time = None
        # 不再使用默认地址，所有服务地址都从服务发现获取
        self.service_urls = {
            'route_plan': None,
            'navi_control': None,
            'sonar': None,
            'target_recognize': None,
            'target_hit': None
        }
        # 保存端点详细信息（节点、实例等）
        self.service_endpoints = {
            'route_plan': None,
            'navi_control': None,
            'sonar': None,
            'target_recognize': None,
            'target_hit': None
        }
        # 必须从 Plum Controller 发现服务地址
        self.discover_services()
    
    def discover_services(self):
        """从 Plum Controller 发现服务地址"""
        try:
            # 使用 /v1/discovery/one 获取单个端点（lazy 策略优先保持缓存）
            for service_key, service_name in SERVICE_NAMES.items():
                try:
                    response = requests.get(
                        f"{CONTROLLER_URL}/v1/discovery/one",
                        params={'service': service_name, 'strategy': 'lazy'},
                        timeout=3
                    )
                    if response.status_code == 200:
                        endpoint = response.json()
                        if not isinstance(endpoint, dict):
                            print(f"[SimDecision] 警告: 服务 {service_name} 返回未知格式: {endpoint}")
                            continue
                        
                        ip = endpoint.get('ip', 'localhost')
                        port = endpoint.get('port', 0)
                        protocol = endpoint.get('protocol', 'http')
                        
                        if port:
                            self.service_urls[service_key] = f"{protocol}://{ip}:{port}"
                            self.service_endpoints[service_key] = {
                                'ip': ip,
                                'port': port,
                                'protocol': protocol,
                                'nodeId': endpoint.get('nodeId', ''),
                                'instanceId': endpoint.get('instanceId', ''),
                                'serviceName': endpoint.get('serviceName', service_name),
                                'healthy': endpoint.get('healthy', False)
                            }
                            print(f"[SimDecision] 发现服务 {service_name}: {self.service_urls[service_key]} "
                                  f"(节点: {endpoint.get('nodeId', 'N/A')}, 实例: {endpoint.get('instanceId', 'N/A')}, "
                                  f"healthy={endpoint.get('healthy', False)})")
                        else:
                            print(f"[SimDecision] 警告: 服务 {service_name} 返回的端口无效: {endpoint}")
                    elif response.status_code == 404:
                        print(f"[SimDecision] 提示: 服务 {service_name} 当前未发现可用端点 (404)")
                    else:
                        print(f"[SimDecision] 警告: 服务发现接口返回 {response.status_code} ({service_name}) - {response.text}")
                except Exception as e:
                    print(f"[SimDecision] 错误: 无法发现服务 {service_name}: {e}")
        except Exception as e:
            print(f"[SimDecision] 服务发现失败: {e}")
    
    def call_route_plan(self, point1, point2, obstacle=None):
        """调用航路规划服务"""
        if not self.service_urls['route_plan']:
            raise Exception("服务 'planRoute1' 未从服务发现获取到地址，请确保服务已注册并运行")
        try:
            url = f"{self.service_urls['route_plan']}/planRoute1"
            data = {
                "point1": point1,
                "point2": point2,
                "obstacle": obstacle or {"polygon": []}
            }
            
            response = requests.post(url, json=data, timeout=REQUEST_TIMEOUT)
            response.raise_for_status()
            result = response.json()
            
            self.results['route_plan'] = {
                'success': result.get('success', False),
                'algorithm': result.get('algorithm', ''),
                'route': result.get('route', []),
                'timestamp': time.time()
            }
            return self.results['route_plan']
        except Exception as e:
            error_msg = str(e)
            self.errors['route_plan'] = error_msg
            self.results['route_plan'] = {
                'success': False,
                'error': error_msg,
                'timestamp': time.time()
            }
            return self.results['route_plan']
    
    def call_navi_control(self, route):
        """调用航控服务"""
        if not self.service_urls['navi_control']:
            raise Exception("服务 'controlUSV' 未从服务发现获取到地址，请确保服务已注册并运行")
        try:
            url = f"{self.service_urls['navi_control']}/controlUSV"
            data = {
                "route": route
            }
            
            response = requests.post(url, json=data, timeout=REQUEST_TIMEOUT)
            response.raise_for_status()
            result = response.json()
            
            self.results['navi_control'] = {
                'success': result.get('success', False),
                'message': result.get('message', ''),
                'waypoints_count': result.get('waypoints_count', 0),
                'status': result.get('status', ''),
                'timestamp': time.time()
            }
            return self.results['navi_control']
        except Exception as e:
            error_msg = str(e)
            self.errors['navi_control'] = error_msg
            self.results['navi_control'] = {
                'success': False,
                'error': error_msg,
                'timestamp': time.time()
            }
            return self.results['navi_control']
    
    def call_sonar(self):
        """调用声纳探测服务"""
        if not self.service_urls['sonar']:
            raise Exception("服务 'detectTarget' 未从服务发现获取到地址，请确保服务已注册并运行")
        try:
            url = f"{self.service_urls['sonar']}/detectTarget"
            
            response = requests.get(url, timeout=REQUEST_TIMEOUT)
            response.raise_for_status()
            result = response.json()
            
            self.results['sonar'] = {
                'success': result.get('success', False),
                'message': result.get('message', ''),
                'target_count': result.get('target_count', 0),
                'targets': result.get('targets', []),
                'timestamp': result.get('timestamp', time.time())
            }
            return self.results['sonar']
        except Exception as e:
            error_msg = str(e)
            self.errors['sonar'] = error_msg
            self.results['sonar'] = {
                'success': False,
                'error': error_msg,
                'timestamp': time.time()
            }
            return self.results['sonar']
    
    def call_target_recognize(self, image_path):
        """调用目标识别服务"""
        if not self.service_urls['target_recognize']:
            raise Exception("服务 'recognizeTarget' 未从服务发现获取到地址，请确保服务已注册并运行")
        try:
            url = f"{self.service_urls['target_recognize']}/recognizeTarget"
            data = {
                "image_path": image_path
            }
            
            response = requests.post(url, json=data, timeout=REQUEST_TIMEOUT)
            response.raise_for_status()
            result = response.json()
            
            self.results['target_recognize'] = {
                'success': result.get('success', False),
                'message': result.get('message', ''),
                'image_path': result.get('image_path', ''),
                'target_type': result.get('target_type', ''),
                'size': result.get('size', ''),
                'confidence': result.get('confidence', 0.0),
                'recognize_time': result.get('recognize_time', time.time())
            }
            return self.results['target_recognize']
        except Exception as e:
            error_msg = str(e)
            self.errors['target_recognize'] = error_msg
            self.results['target_recognize'] = {
                'success': False,
                'error': error_msg,
                'timestamp': time.time()
            }
            return self.results['target_recognize']
    
    def call_target_hit(self, target_id, longitude, latitude):
        """调用目标打击服务"""
        if not self.service_urls['target_hit']:
            raise Exception("服务 'hitTarget' 未从服务发现获取到地址，请确保服务已注册并运行")
        try:
            # 确保 URL 格式正确，避免多余的斜杠
            base_url = self.service_urls['target_hit'].rstrip('/')
            url = f"{base_url}/hitTarget"
            data = {
                "id": target_id,
                "longitude": longitude,
                "latitude": latitude
            }
            
            print(f"[SimDecision] 调用目标打击服务: {url}, 数据: {data}")
            print(f"[SimDecision] 服务地址来源: {self.service_urls['target_hit']}")
            response = requests.post(url, json=data, timeout=REQUEST_TIMEOUT)
            
            # 检查响应内容类型
            content_type = response.headers.get('Content-Type', '')
            print(f"[SimDecision] 目标打击响应状态: {response.status_code}, Content-Type: {content_type}")
            
            # 如果返回的不是 JSON，记录响应内容的前500个字符
            if 'application/json' not in content_type:
                response_text = response.text[:500]
                print(f"[SimDecision] 目标打击服务返回非JSON响应: {response_text}")
                raise Exception(f"服务返回非JSON响应 (状态码: {response.status_code}): {response_text[:200]}")
            
            response.raise_for_status()
            result = response.json()
            
            self.results['target_hit'] = {
                'success': result.get('success', False),
                'message': result.get('message', ''),
                'target_id': result.get('target_id', target_id),
                'longitude': result.get('longitude', longitude),
                'latitude': result.get('latitude', latitude),
                'hit_time': result.get('hit_time', time.time()),
                'damage': result.get('damage', ''),
                'status': result.get('status', '')
            }
            return self.results['target_hit']
        except requests.exceptions.JSONDecodeError as e:
            error_msg = f"JSON解析失败: {str(e)}。响应内容: {response.text[:200] if 'response' in locals() else 'N/A'}"
            print(f"[SimDecision] 目标打击服务调用错误: {error_msg}")
            self.errors['target_hit'] = error_msg
            self.results['target_hit'] = {
                'success': False,
                'error': error_msg,
                'timestamp': time.time()
            }
            return self.results['target_hit']
        except Exception as e:
            error_msg = str(e)
            print(f"[SimDecision] 目标打击服务调用异常: {error_msg}")
            self.errors['target_hit'] = error_msg
            self.results['target_hit'] = {
                'success': False,
                'error': error_msg,
                'timestamp': time.time()
            }
            return self.results['target_hit']
    
    def execute_full_workflow(self, point1, point2, obstacle=None):
        """执行完整的决策流程"""
        self.start_time = time.time()
        workflow_status = {
            'step': 0,
            'total_steps': 4,
            'status': 'running',
            'steps': []
        }
        
        # 步骤1：航路规划
        workflow_status['step'] = 1
        workflow_status['steps'].append({
            'name': '航路规划',
            'status': 'running',
            'start_time': time.time()
        })
        route_result = self.call_route_plan(point1, point2, obstacle)
        workflow_status['steps'][-1]['status'] = 'success' if route_result['success'] else 'failed'
        workflow_status['steps'][-1]['end_time'] = time.time()
        workflow_status['steps'][-1]['result'] = route_result
        
        if not route_result['success']:
            workflow_status['status'] = 'failed'
            self.end_time = time.time()
            return workflow_status
        
        # 步骤2：启动航控
        workflow_status['step'] = 2
        workflow_status['steps'].append({
            'name': '启动航控',
            'status': 'running',
            'start_time': time.time()
        })
        navi_result = self.call_navi_control(route_result.get('route', []))
        workflow_status['steps'][-1]['status'] = 'success' if navi_result['success'] else 'failed'
        workflow_status['steps'][-1]['end_time'] = time.time()
        workflow_status['steps'][-1]['result'] = navi_result
        
        if not navi_result['success']:
            workflow_status['status'] = 'failed'
            self.end_time = time.time()
            return workflow_status
        
        # 步骤3：声纳探测
        workflow_status['step'] = 3
        workflow_status['steps'].append({
            'name': '声纳探测',
            'status': 'running',
            'start_time': time.time()
        })
        sonar_result = self.call_sonar()
        workflow_status['steps'][-1]['status'] = 'success' if sonar_result['success'] else 'failed'
        workflow_status['steps'][-1]['end_time'] = time.time()
        workflow_status['steps'][-1]['result'] = sonar_result
        
        if not sonar_result['success']:
            workflow_status['status'] = 'failed'
            self.end_time = time.time()
            return workflow_status
        
        # 步骤4：目标识别（对每个检测到的目标进行识别）
        workflow_status['step'] = 4
        targets = sonar_result.get('targets', [])
        target_recognize_results = []
        
        for i, target in enumerate(targets):
            image_path = target.get('image_path', '')
            if image_path:
                step_name = f'目标识别 (目标 {target.get("id", i+1)})'
                workflow_status['steps'].append({
                    'name': step_name,
                    'status': 'running',
                    'start_time': time.time()
                })
                try:
                    recognize_result = self.call_target_recognize(image_path)
                    workflow_status['steps'][-1]['status'] = 'success' if recognize_result['success'] else 'failed'
                    workflow_status['steps'][-1]['end_time'] = time.time()
                    workflow_status['steps'][-1]['result'] = recognize_result
                    target_recognize_results.append(recognize_result)
                except Exception as e:
                    workflow_status['steps'][-1]['status'] = 'failed'
                    workflow_status['steps'][-1]['end_time'] = time.time()
                    workflow_status['steps'][-1]['result'] = {'success': False, 'error': str(e)}
                    target_recognize_results.append({'success': False, 'error': str(e)})
        
        # 如果所有目标识别都成功，继续目标打击
        all_recognize_success = all(r.get('success', False) for r in target_recognize_results) if target_recognize_results else True
        if not all_recognize_success:
            workflow_status['status'] = 'failed'
            self.end_time = time.time()
            workflow_status['total_time'] = self.end_time - self.start_time
            return workflow_status
        
        # 步骤5：目标打击（对每个识别成功的目标进行打击）
        workflow_status['step'] = 5
        target_hit_results = []
        
        # 将目标识别结果与声纳探测目标匹配
        for i, target in enumerate(targets):
            image_path = target.get('image_path', '')
            if image_path:
                # 找到对应的识别结果
                recognize_result = None
                for r in target_recognize_results:
                    if r.get('success') and r.get('image_path') == image_path:
                        recognize_result = r
                        break
                
                # 如果识别成功，进行目标打击
                if recognize_result and recognize_result.get('success'):
                    target_id = target.get('id', i + 1)
                    longitude = target.get('longitude', 0.0)
                    latitude = target.get('latitude', 0.0)
                    
                    step_name = f'目标打击 (目标 {target_id})'
                    workflow_status['steps'].append({
                        'name': step_name,
                        'status': 'running',
                        'start_time': time.time()
                    })
                    try:
                        hit_result = self.call_target_hit(target_id, longitude, latitude)
                        workflow_status['steps'][-1]['status'] = 'success' if hit_result['success'] else 'failed'
                        workflow_status['steps'][-1]['end_time'] = time.time()
                        workflow_status['steps'][-1]['result'] = hit_result
                        target_hit_results.append(hit_result)
                    except Exception as e:
                        workflow_status['steps'][-1]['status'] = 'failed'
                        workflow_status['steps'][-1]['end_time'] = time.time()
                        workflow_status['steps'][-1]['result'] = {'success': False, 'error': str(e)}
                        target_hit_results.append({'success': False, 'error': str(e)})
        
        # 如果所有目标打击都成功，工作流成功
        all_hit_success = all(r.get('success', False) for r in target_hit_results) if target_hit_results else True
        workflow_status['status'] = 'success' if all_hit_success else 'failed'
        self.end_time = time.time()
        workflow_status['total_time'] = self.end_time - self.start_time
        
        return workflow_status


@app.route('/')
def index():
    """主页面"""
    return render_template('index.html')


# 全局 ServiceCaller 实例，用于跨请求共享服务发现信息
_global_caller = None

def get_service_caller():
    """获取全局 ServiceCaller 实例"""
    global _global_caller
    if _global_caller is None:
        _global_caller = ServiceCaller()
    return _global_caller

@app.route('/api/call_route_plan', methods=['POST'])
def call_route_plan_api():
    """调用航路规划服务"""
    try:
        data = request.json
        point1 = data.get('point1', {})
        point2 = data.get('point2', {})
        obstacle = data.get('obstacle', None)
        
        if not point1.get('longitude') or not point1.get('latitude'):
            return jsonify({'success': False, 'error': 'point1 必须包含 longitude 和 latitude'}), 400
        if not point2.get('longitude') or not point2.get('latitude'):
            return jsonify({'success': False, 'error': 'point2 必须包含 longitude 和 latitude'}), 400
        
        caller = get_service_caller()
        # 重新发现服务，确保端点信息是最新的
        caller.discover_services()
        result = caller.call_route_plan(point1, point2, obstacle)
        
        # 获取端点信息，如果为空则尝试从服务发现重新获取
        endpoint_info = caller.service_endpoints.get('route_plan')
        if not endpoint_info:
            # 尝试通过 /v1/discovery/one 重新获取端点信息
            try:
                response = requests.get(
                    f"{CONTROLLER_URL}/v1/discovery/one",
                    params={'service': SERVICE_NAMES['route_plan'], 'strategy': 'lazy'},
                    timeout=3
                )
                if response.status_code == 200:
                    ep = response.json()
                    if isinstance(ep, dict):
                        endpoint_info = {
                            'ip': ep.get('ip', ''),
                            'port': ep.get('port', 0),
                            'protocol': ep.get('protocol', 'http'),
                            'nodeId': ep.get('nodeId', ''),
                            'instanceId': ep.get('instanceId', ''),
                            'healthy': ep.get('healthy', False)
                        }
                        caller.service_endpoints['route_plan'] = endpoint_info
                elif response.status_code == 404:
                    print("[SimDecision] [route_plan] 服务发现返回 404，暂无可用端点")
                else:
                    print(f"[SimDecision] [route_plan] 服务发现接口返回 {response.status_code}: {response.text}")
            except Exception as e:
                print(f"[SimDecision] 无法获取端点信息: {e}")
        
        # 如果仍然没有端点信息，但服务调用成功（说明 service_urls 有地址），从 URL 构造
        if not endpoint_info and caller.service_urls.get('route_plan'):
            from urllib.parse import urlparse
            try:
                parsed = urlparse(caller.service_urls['route_plan'])
                # 如果没有显式端口，使用协议默认端口
                port = parsed.port
                if not port:
                    port = 80 if parsed.scheme == 'http' else (443 if parsed.scheme == 'https' else 0)
                endpoint_info = {
                    'ip': parsed.hostname or 'localhost',
                    'port': port,
                    'protocol': parsed.scheme or 'http',
                    'nodeId': '',
                    'instanceId': '',
                    'healthy': False
                }
                # 保存到 service_endpoints，避免下次重新构造
                caller.service_endpoints['route_plan'] = endpoint_info
                print(f"[SimDecision] [route_plan] 从 service_urls 构造端点信息: {endpoint_info}")
            except Exception as e:
                print(f"[SimDecision] 无法从 URL 构造端点信息: {e}")
        
        return jsonify({
            'success': result.get('success', False),
            'result': result,
            'service_endpoint': endpoint_info if endpoint_info else None
        })
    except Exception as e:
        return jsonify({'success': False, 'error': str(e)}), 500

@app.route('/api/call_navi_control', methods=['POST'])
def call_navi_control_api():
    """调用航控服务"""
    try:
        data = request.json
        route = data.get('route', [])
        
        if not route:
            return jsonify({'success': False, 'error': 'route 不能为空'}), 400
        
        caller = get_service_caller()
        # 重新发现服务，确保端点信息是最新的
        caller.discover_services()
        result = caller.call_navi_control(route)
        
        # 获取端点信息，如果为空则尝试从服务发现重新获取
        endpoint_info = caller.service_endpoints.get('navi_control')
        if not endpoint_info:
            try:
                response = requests.get(
                    f"{CONTROLLER_URL}/v1/discovery/one",
                    params={'service': SERVICE_NAMES['navi_control'], 'strategy': 'lazy'},
                    timeout=3
                )
                if response.status_code == 200:
                    ep = response.json()
                    if isinstance(ep, dict):
                        endpoint_info = {
                            'ip': ep.get('ip', ''),
                            'port': ep.get('port', 0),
                            'protocol': ep.get('protocol', 'http'),
                            'nodeId': ep.get('nodeId', ''),
                            'instanceId': ep.get('instanceId', ''),
                            'healthy': ep.get('healthy', False)
                        }
                        caller.service_endpoints['navi_control'] = endpoint_info
                elif response.status_code == 404:
                    print("[SimDecision] [navi_control] 服务发现返回 404，暂无可用端点")
                else:
                    print(f"[SimDecision] [navi_control] 服务发现接口返回 {response.status_code}: {response.text}")
            except Exception as e:
                print(f"[SimDecision] 无法获取端点信息: {e}")
        
        # 如果仍然没有端点信息，但服务调用成功（说明 service_urls 有地址），从 URL 构造
        if not endpoint_info and caller.service_urls.get('navi_control'):
            from urllib.parse import urlparse
            try:
                parsed = urlparse(caller.service_urls['navi_control'])
                endpoint_info = {
                    'ip': parsed.hostname or 'localhost',
                    'port': parsed.port or 0,
                    'protocol': parsed.scheme or 'http',
                    'nodeId': '',
                    'instanceId': '',
                    'healthy': False
                }
                # 保存到 service_endpoints，避免下次重新构造
                caller.service_endpoints['navi_control'] = endpoint_info
                print(f"[SimDecision] [navi_control] 从 service_urls 构造端点信息: {endpoint_info}")
            except Exception as e:
                print(f"[SimDecision] 无法从 URL 构造端点信息: {e}")
        
        return jsonify({
            'success': result.get('success', False),
            'result': result,
            'service_endpoint': endpoint_info if endpoint_info else None
        })
    except Exception as e:
        return jsonify({'success': False, 'error': str(e)}), 500

@app.route('/api/call_sonar', methods=['POST'])
def call_sonar_api():
    """调用声纳探测服务"""
    try:
        caller = get_service_caller()
        # 重新发现服务，确保端点信息是最新的
        caller.discover_services()
        result = caller.call_sonar()
        
        # 获取端点信息，如果为空则尝试从服务发现重新获取
        endpoint_info = caller.service_endpoints.get('sonar')
        if not endpoint_info:
            try:
                response = requests.get(
                    f"{CONTROLLER_URL}/v1/discovery/one",
                    params={'service': SERVICE_NAMES['sonar'], 'strategy': 'lazy'},
                    timeout=3
                )
                if response.status_code == 200:
                    ep = response.json()
                    if isinstance(ep, dict):
                        endpoint_info = {
                            'ip': ep.get('ip', ''),
                            'port': ep.get('port', 0),
                            'protocol': ep.get('protocol', 'http'),
                            'nodeId': ep.get('nodeId', ''),
                            'instanceId': ep.get('instanceId', ''),
                            'healthy': ep.get('healthy', False)
                        }
                        caller.service_endpoints['sonar'] = endpoint_info
                elif response.status_code == 404:
                    print("[SimDecision] [sonar] 服务发现返回 404，暂无可用端点")
                else:
                    print(f"[SimDecision] [sonar] 服务发现接口返回 {response.status_code}: {response.text}")
            except Exception as e:
                print(f"[SimDecision] 无法获取端点信息: {e}")
        
        # 如果仍然没有端点信息，但服务调用成功（说明 service_urls 有地址），从 URL 构造
        if not endpoint_info and caller.service_urls.get('sonar'):
            from urllib.parse import urlparse
            try:
                parsed = urlparse(caller.service_urls['sonar'])
                endpoint_info = {
                    'ip': parsed.hostname or 'localhost',
                    'port': parsed.port or 0,
                    'protocol': parsed.scheme or 'http',
                    'nodeId': '',
                    'instanceId': '',
                    'healthy': False
                }
                # 保存到 service_endpoints，避免下次重新构造
                caller.service_endpoints['sonar'] = endpoint_info
                print(f"[SimDecision] [sonar] 从 service_urls 构造端点信息: {endpoint_info}")
            except Exception as e:
                print(f"[SimDecision] 无法从 URL 构造端点信息: {e}")
        
        return jsonify({
            'success': result.get('success', False),
            'result': result,
            'service_endpoint': endpoint_info if endpoint_info else None
        })
    except Exception as e:
        return jsonify({'success': False, 'error': str(e)}), 500

@app.route('/api/call_target_recognize', methods=['POST'])
def call_target_recognize_api():
    """调用目标识别服务"""
    try:
        data = request.json
        image_path = data.get('image_path', '')
        
        if not image_path:
            return jsonify({'success': False, 'error': 'image_path 不能为空'}), 400
        
        caller = get_service_caller()
        # 重新发现服务，确保端点信息是最新的
        caller.discover_services()
        result = caller.call_target_recognize(image_path)
        
        # 获取端点信息，如果为空则尝试从服务发现重新获取
        endpoint_info = caller.service_endpoints.get('target_recognize')
        if not endpoint_info:
            try:
                response = requests.get(
                    f"{CONTROLLER_URL}/v1/discovery/one",
                    params={'service': SERVICE_NAMES['target_recognize'], 'strategy': 'lazy'},
                    timeout=3
                )
                if response.status_code == 200:
                    ep = response.json()
                    if isinstance(ep, dict):
                        endpoint_info = {
                            'ip': ep.get('ip', ''),
                            'port': ep.get('port', 0),
                            'protocol': ep.get('protocol', 'http'),
                            'nodeId': ep.get('nodeId', ''),
                            'instanceId': ep.get('instanceId', ''),
                            'healthy': ep.get('healthy', False)
                        }
                        caller.service_endpoints['target_recognize'] = endpoint_info
                elif response.status_code == 404:
                    print("[SimDecision] [target_recognize] 服务发现返回 404，暂无可用端点")
                else:
                    print(f"[SimDecision] [target_recognize] 服务发现接口返回 {response.status_code}: {response.text}")
            except Exception as e:
                print(f"[SimDecision] 无法获取端点信息: {e}")
        
        # 如果仍然没有端点信息，但服务调用成功（说明 service_urls 有地址），从 URL 构造
        if not endpoint_info and caller.service_urls.get('target_recognize'):
            from urllib.parse import urlparse
            try:
                parsed = urlparse(caller.service_urls['target_recognize'])
                # 如果没有显式端口，使用协议默认端口
                port = parsed.port
                if not port:
                    port = 80 if parsed.scheme == 'http' else (443 if parsed.scheme == 'https' else 0)
                endpoint_info = {
                    'ip': parsed.hostname or 'localhost',
                    'port': port,
                    'protocol': parsed.scheme or 'http',
                    'nodeId': '',
                    'instanceId': '',
                    'healthy': False
                }
                # 保存到 service_endpoints，避免下次重新构造
                caller.service_endpoints['target_recognize'] = endpoint_info
                print(f"[SimDecision] [target_recognize] 从 service_urls 构造端点信息: {endpoint_info}")
            except Exception as e:
                print(f"[SimDecision] 无法从 URL 构造端点信息: {e}")
        
        return jsonify({
            'success': result.get('success', False),
            'result': result,
            'service_endpoint': endpoint_info if endpoint_info else None
        })
    except Exception as e:
        return jsonify({'success': False, 'error': str(e)}), 500

@app.route('/api/call_target_hit', methods=['POST'])
def call_target_hit_api():
    """调用目标打击服务"""
    try:
        data = request.json
        target_id = data.get('id', 0)
        longitude = data.get('longitude', 0.0)
        latitude = data.get('latitude', 0.0)
        
        if target_id == 0:
            return jsonify({'success': False, 'error': 'target_id 不能为空'}), 400
        
        caller = get_service_caller()
        # 重新发现服务，确保端点信息是最新的
        caller.discover_services()
        result = caller.call_target_hit(target_id, longitude, latitude)
        
        # 获取端点信息，如果为空则尝试从服务发现重新获取
        endpoint_info = caller.service_endpoints.get('target_hit')
        if not endpoint_info:
            try:
                response = requests.get(
                    f"{CONTROLLER_URL}/v1/discovery/one",
                    params={'service': SERVICE_NAMES['target_hit'], 'strategy': 'lazy'},
                    timeout=3
                )
                if response.status_code == 200:
                    ep = response.json()
                    if isinstance(ep, dict):
                        endpoint_info = {
                            'ip': ep.get('ip', ''),
                            'port': ep.get('port', 0),
                            'protocol': ep.get('protocol', 'http'),
                            'nodeId': ep.get('nodeId', ''),
                            'instanceId': ep.get('instanceId', ''),
                            'healthy': ep.get('healthy', False)
                        }
                        caller.service_endpoints['target_hit'] = endpoint_info
                elif response.status_code == 404:
                    print("[SimDecision] [target_hit] 服务发现返回 404，暂无可用端点")
                else:
                    print(f"[SimDecision] [target_hit] 服务发现接口返回 {response.status_code}: {response.text}")
            except Exception as e:
                print(f"[SimDecision] 无法获取端点信息: {e}")
        
        # 如果仍然没有端点信息，但服务调用成功（说明 service_urls 有地址），从 URL 构造
        if not endpoint_info and caller.service_urls.get('target_hit'):
            from urllib.parse import urlparse
            try:
                parsed = urlparse(caller.service_urls['target_hit'])
                # 如果没有显式端口，使用协议默认端口
                port = parsed.port
                if not port:
                    port = 80 if parsed.scheme == 'http' else (443 if parsed.scheme == 'https' else 0)
                endpoint_info = {
                    'ip': parsed.hostname or 'localhost',
                    'port': port,
                    'protocol': parsed.scheme or 'http',
                    'nodeId': '',
                    'instanceId': '',
                    'healthy': False
                }
                # 保存到 service_endpoints，避免下次重新构造
                caller.service_endpoints['target_hit'] = endpoint_info
                print(f"[SimDecision] [target_hit] 从 service_urls 构造端点信息: {endpoint_info}")
            except Exception as e:
                print(f"[SimDecision] 无法从 URL 构造端点信息: {e}")
        
        return jsonify({
            'success': result.get('success', False),
            'result': result,
            'service_endpoint': endpoint_info if endpoint_info else None
        })
    except Exception as e:
        return jsonify({'success': False, 'error': str(e)}), 500

@app.route('/api/execute', methods=['POST'])
def execute_workflow():
    """执行决策流程（保留用于兼容）"""
    try:
        data = request.json
        point1 = data.get('point1', {})
        point2 = data.get('point2', {})
        obstacle = data.get('obstacle', None)
        
        # 验证输入
        if not point1.get('longitude') or not point1.get('latitude'):
            return jsonify({'success': False, 'error': 'point1 必须包含 longitude 和 latitude'}), 400
        if not point2.get('longitude') or not point2.get('latitude'):
            return jsonify({'success': False, 'error': 'point2 必须包含 longitude 和 latitude'}), 400
        
        caller = ServiceCaller()
        workflow_status = caller.execute_full_workflow(point1, point2, obstacle)
        
        return jsonify({
            'success': workflow_status['status'] == 'success',
            'workflow': workflow_status,
            'results': caller.results or {},  # 确保始终是对象
            'errors': caller.errors or {},  # 确保始终是对象
            'service_endpoints': caller.service_endpoints or {}  # 确保始终是对象
        })
    except Exception as e:
        return jsonify({'success': False, 'error': str(e)}), 500


@app.route('/api/status')
def get_status():
    """获取服务状态（直接从服务发现获取健康状态）"""
    status = {
        'route_plan': False,
        'navi_control': False,
        'sonar': False,
        'target_recognize': False,
        'target_hit': False,
        'messages': {
            'route_plan': '',
            'navi_control': '',
            'sonar': '',
            'target_recognize': '',
            'target_hit': ''
        }
    }
    
    # 从 Plum Controller 服务发现获取服务状态
    for service_key, service_name in SERVICE_NAMES.items():
        try:
            response = requests.get(
                f"{CONTROLLER_URL}/v1/discovery/one",
                params={'service': service_name, 'strategy': 'lazy'},
                timeout=3
            )
            if response.status_code == 200:
                endpoint = response.json()
                if isinstance(endpoint, dict):
                    is_healthy = endpoint.get('healthy', False)
                    status[service_key] = is_healthy
                    ip = endpoint.get('ip', 'N/A')
                    port = endpoint.get('port', 'N/A')
                    node_id = endpoint.get('nodeId', 'N/A')
                    if is_healthy:
                        status['messages'][service_key] = f'服务在线 ({ip}:{port}, 节点: {node_id})'
                    else:
                        status['messages'][service_key] = f'服务不健康 ({ip}:{port}, 节点: {node_id})'
                else:
                    status['messages'][service_key] = f'Controller 返回未知格式: {endpoint}'
            elif response.status_code == 404:
                status['messages'][service_key] = f'服务未发现，请确保服务 "{service_name}" 已注册'
            else:
                status['messages'][service_key] = f'无法连接到 Controller: HTTP {response.status_code}'
        except requests.exceptions.ConnectionError:
            status['messages'][service_key] = f'无法连接到 Controller: {CONTROLLER_URL}'
        except requests.exceptions.Timeout:
            status['messages'][service_key] = '连接 Controller 超时'
        except Exception as e:
            status['messages'][service_key] = f'错误: {str(e)}'
    
    return jsonify(status)


if __name__ == '__main__':
    port = int(os.getenv('PORT', 3000))
    debug = os.getenv('DEBUG', 'False').lower() == 'true'
    app.run(host='0.0.0.0', port=port, debug=debug)

