export class HttpRequest {
  private baseUrl: string;
  private options: Record<string, any>;

  /**
   * 构造函数
   * @param baseUrl 基础 URL
   */
  constructor(base: string) {
    this.baseUrl = base;
    this.options = {
      timeout: 60,
    };
  }

  /**
   * 发送 fetch 请求
   * @param endpoint 请求的端点路径
   * @param options 请求参数
   * @returns 返回解析后的 JSON 数据
   */
  fetch(endpoint: string, options?: Record<string, any>): Promise<Response> {
    return fetch(`${this.baseUrl}${endpoint}`, {
      headers: {
        'Content-Type': 'application/json'
      },
      ...this.options,
      ...options,
    }) as Promise<Response>;
  }

  /**
   * 发送 GET 请求
   * @param endpoint 请求的端点路径
   * @param params 请求参数
   * @returns 返回解析后的 JSON 数据
   */
  async get<T>(endpoint: string, params?: Record<string, any>): Promise<T> {
    const url = `${this.baseUrl}${endpoint}`;
    const param = new URLSearchParams(params);
    const query = params ? `?${param.toString()}` : '';
    const response = await fetch(`${url}${query}`, {
      ...this.options,
      method: 'GET',
      headers: { 'Content-Type': 'application/json' },
    });
    return response.json() as Promise<T>;
  }

  /**
   * 发送 POST 请求
   * @param endpoint 请求的端点路径
   * @param data 请求体数据
   * @returns 返回解析后的 JSON 数据
   */
  async post<T>(endpoint: string, data?: Record<string, any>): Promise<T> {
    const url = `${this.baseUrl}${endpoint}`;
    const response = await fetch(url, {
      ...this.options,
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    return response.json() as Promise<T>;
  }

  /**
   * 发送 PUT 请求
   * @param endpoint 请求的端点路径
   * @param data 请求体数据
   * @returns 返回解析后的 JSON 数据
   */
  async put<T>(endpoint: string, data?: Record<string, any>): Promise<T> {
    const url = `${this.baseUrl}${endpoint}`;
    const response = await fetch(url, {
      ...this.options,
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    return response.json() as Promise<T>;
  }

  /**
   * 发送 DELETE 请求
   * @param endpoint 请求的端点路径
   * @returns 返回解析后的 JSON 数据
   */
  async delete<T>(endpoint: string): Promise<T> {
    const url = `${this.baseUrl}${endpoint}`;
    const response = await fetch(url, {
      ...this.options,
      method: 'DELETE',
      headers: { 'Content-Type': 'application/json' },
    });
    return response.json() as Promise<T>;
  }
}