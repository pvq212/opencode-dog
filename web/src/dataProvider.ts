import type { DataProvider } from 'react-admin';

const API_URL = '/api';

function getHeaders(): HeadersInit {
  const token = localStorage.getItem('token');
  return {
    'Content-Type': 'application/json',
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
  };
}

async function handleResponse(response: Response) {
  if (!response.ok) {
    const body = await response.json().catch(() => ({ error: response.statusText }));
    const error = new Error(body.error || response.statusText) as Error & { status: number };
    error.status = response.status;
    throw error;
  }
  const text = await response.text();
  return text ? JSON.parse(text) : {};
}

function resourceToEndpoint(resource: string, params?: Record<string, unknown>): string {
  switch (resource) {
    case 'projects':
      return `${API_URL}/projects`;
    case 'ssh-keys':
      return `${API_URL}/ssh-keys`;
    case 'providers':
      return `${API_URL}/providers/${params?.projectId || ''}`;
    case 'keywords':
      return `${API_URL}/keywords/${params?.projectId || ''}`;
    case 'tasks':
      return `${API_URL}/tasks`;
    case 'settings':
      return `${API_URL}/settings`;
    case 'mcp-servers':
      return `${API_URL}/mcp-servers`;
    case 'users':
      return `${API_URL}/users`;
    default:
      return `${API_URL}/${resource}`;
  }
}

const dataProvider: DataProvider = {
  getList: async (resource, params) => {
    const { page, perPage } = params.pagination || { page: 1, perPage: 25 };
    const { field, order } = params.sort || { field: 'id', order: 'ASC' };
    const filter = params.filter || {};

    if (resource === 'tasks') {
      const query = new URLSearchParams({
        limit: String(perPage),
        offset: String((page - 1) * perPage),
        ...(filter.status ? { status: filter.status } : {}),
        ...(filter.provider_type ? { provider_type: filter.provider_type } : {}),
      });
      const response = await fetch(`${API_URL}/tasks?${query}`, { headers: getHeaders() });
      const data = await handleResponse(response);
      return {
        data: data.tasks || [],
        total: data.total || 0,
      };
    }

    if (resource === 'providers' && filter.projectId) {
      const response = await fetch(`${API_URL}/providers/${filter.projectId}`, { headers: getHeaders() });
      const data = await handleResponse(response);
      const list = Array.isArray(data) ? data : data.data || [];
      return { data: list, total: list.length };
    }

    if (resource === 'keywords' && filter.projectId) {
      const response = await fetch(`${API_URL}/keywords/${filter.projectId}`, { headers: getHeaders() });
      const data = await handleResponse(response);
      const list = Array.isArray(data) ? data : data.data || [];
      return { data: list.map((k: Record<string, unknown>, i: number) => ({ id: i, ...k })), total: list.length };
    }

    if (resource === 'settings') {
      const response = await fetch(`${API_URL}/settings`, { headers: getHeaders() });
      const data = await handleResponse(response);
      const list = Array.isArray(data) ? data : data.data || Object.entries(data).map(([key, value]) => ({ id: key, key, value }));
      return { data: list, total: list.length };
    }

    const endpoint = resourceToEndpoint(resource);
    const response = await fetch(endpoint, { headers: getHeaders() });
    const data = await handleResponse(response);
    let list = Array.isArray(data) ? data : data.data || [];

    // Client-side sort
    list = [...list].sort((a: Record<string, unknown>, b: Record<string, unknown>) => {
      const aVal = a[field];
      const bVal = b[field];
      if (aVal == null) return 1;
      if (bVal == null) return -1;
      if (typeof aVal === 'string' && typeof bVal === 'string') {
        return order === 'ASC' ? aVal.localeCompare(bVal) : bVal.localeCompare(aVal);
      }
      return order === 'ASC' ? Number(aVal) - Number(bVal) : Number(bVal) - Number(aVal);
    });

    const total = list.length;
    const start = (page - 1) * perPage;
    const paged = list.slice(start, start + perPage);

    return { data: paged, total };
  },

  getOne: async (resource, params) => {
    if (resource === 'settings') {
      const response = await fetch(`${API_URL}/settings/${params.id}`, { headers: getHeaders() });
      const data = await handleResponse(response);
      return { data: { id: data.key || params.id, ...data } };
    }
    const endpoint = resourceToEndpoint(resource);
    const response = await fetch(`${endpoint}/${params.id}`, { headers: getHeaders() });
    const data = await handleResponse(response);
    return { data };
  },

  getMany: async (resource, params) => {
    const results = await Promise.all(
      params.ids.map(async (id) => {
        const endpoint = resourceToEndpoint(resource);
        const response = await fetch(`${endpoint}/${id}`, { headers: getHeaders() });
        return handleResponse(response);
      })
    );
    return { data: results };
  },

  getManyReference: async (resource, params) => {
    const { page, perPage } = params.pagination || { page: 1, perPage: 25 };
    const filter = { ...params.filter, [params.target]: params.id };

    if (resource === 'providers') {
      const projectId = filter.projectId || filter.project_id || params.id;
      const response = await fetch(`${API_URL}/providers/${projectId}`, { headers: getHeaders() });
      const data = await handleResponse(response);
      const list = Array.isArray(data) ? data : data.data || [];
      return { data: list, total: list.length };
    }

    if (resource === 'keywords') {
      const projectId = filter.projectId || filter.project_id || params.id;
      const response = await fetch(`${API_URL}/keywords/${projectId}`, { headers: getHeaders() });
      const data = await handleResponse(response);
      const list = Array.isArray(data) ? data : data.data || [];
      return {
        data: list.map((k: Record<string, unknown>, i: number) => ({ id: i, ...k })),
        total: list.length,
      };
    }

    const endpoint = resourceToEndpoint(resource, filter);
    const response = await fetch(endpoint, { headers: getHeaders() });
    const data = await handleResponse(response);
    const list = Array.isArray(data) ? data : data.data || [];
    const start = (page - 1) * perPage;
    return { data: list.slice(start, start + perPage), total: list.length };
  },

  create: async (resource, params) => {
    if (resource === 'providers' && params.data.projectId) {
      const { projectId, ...rest } = params.data;
      const response = await fetch(`${API_URL}/providers/${projectId}`, {
        method: 'POST',
        headers: getHeaders(),
        body: JSON.stringify(rest),
      });
      const data = await handleResponse(response);
      return { data };
    }

    if (resource === 'settings') {
      const response = await fetch(`${API_URL}/settings`, {
        method: 'PUT',
        headers: getHeaders(),
        body: JSON.stringify({ key: params.data.key, value: params.data.value }),
      });
      const data = await handleResponse(response);
      return { data: { id: params.data.key, ...data } };
    }

    const endpoint = resourceToEndpoint(resource);
    const response = await fetch(endpoint, {
      method: 'POST',
      headers: getHeaders(),
      body: JSON.stringify(params.data),
    });
    const data = await handleResponse(response);
    return { data };
  },

  update: async (resource, params) => {
    if (resource === 'settings') {
      const response = await fetch(`${API_URL}/settings`, {
        method: 'PUT',
        headers: getHeaders(),
        body: JSON.stringify({ key: params.id, value: params.data.value }),
      });
      const data = await handleResponse(response);
      return { data: { id: params.id, ...data } };
    }

    if (resource === 'keywords' && params.data.projectId) {
      const { projectId, keywords } = params.data;
      const response = await fetch(`${API_URL}/keywords/${projectId}`, {
        method: 'PUT',
        headers: getHeaders(),
        body: JSON.stringify(keywords),
      });
      const data = await handleResponse(response);
      return { data: { id: params.id, ...data } };
    }

    const endpoint = resourceToEndpoint(resource);
    const response = await fetch(`${endpoint}/${params.id}`, {
      method: 'PUT',
      headers: getHeaders(),
      body: JSON.stringify(params.data),
    });
    const data = await handleResponse(response);
    return { data };
  },

  updateMany: async (resource, params) => {
    const results = await Promise.all(
      params.ids.map(async (id) => {
        const endpoint = resourceToEndpoint(resource);
        const response = await fetch(`${endpoint}/${id}`, {
          method: 'PUT',
          headers: getHeaders(),
          body: JSON.stringify(params.data),
        });
        await handleResponse(response);
        return id;
      })
    );
    return { data: results };
  },

  delete: async (resource, params) => {
    if (resource === 'providers' && params.previousData?.projectId) {
      const response = await fetch(
        `${API_URL}/providers/${params.previousData.projectId}/${params.id}`,
        { method: 'DELETE', headers: getHeaders() }
      );
      await handleResponse(response);
      return { data: params.previousData } as any;
    }

    const endpoint = resourceToEndpoint(resource);
    const response = await fetch(`${endpoint}/${params.id}`, {
      method: 'DELETE',
      headers: getHeaders(),
    });
    await handleResponse(response);
    return { data: params.previousData } as any;
  },

  deleteMany: async (resource, params) => {
    await Promise.all(
      params.ids.map(async (id) => {
        const endpoint = resourceToEndpoint(resource);
        await fetch(`${endpoint}/${id}`, { method: 'DELETE', headers: getHeaders() });
      })
    );
    return { data: params.ids };
  },
};

// Custom methods for non-standard endpoints
export async function installMcpServer(id: string | number): Promise<void> {
  const response = await fetch(`${API_URL}/mcp-servers/${id}/install`, {
    method: 'POST',
    headers: getHeaders(),
  });
  if (!response.ok) {
    const body = await response.json().catch(() => ({}));
    throw new Error(body.error || 'Install failed');
  }
}

export async function saveKeywords(projectId: string, keywords: Array<{ keyword: string; mode: string }>): Promise<void> {
  const response = await fetch(`${API_URL}/keywords/${projectId}`, {
    method: 'PUT',
    headers: getHeaders(),
    body: JSON.stringify(keywords),
  });
  if (!response.ok) {
    const body = await response.json().catch(() => ({}));
    throw new Error(body.error || 'Save failed');
  }
}

export default dataProvider;
