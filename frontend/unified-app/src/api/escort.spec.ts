/**
 * api/escort 单测。
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import {
  registerEscort,
  updateLocation,
  updateCity,
  createQualification,
  updateQualification,
  deleteQualification,
  addAvailability,
  removeAvailability,
} from './escort';

const requestSpy = vi.fn();
vi.mock('./client', () => ({ request: (...args: unknown[]) => requestSpy(...args) }));

beforeEach(() => {
  requestSpy.mockReset();
  requestSpy.mockResolvedValue({});
});

describe('api/escort · register / location', () => {
  it('registerEscort POSTs city + nickname', async () => {
    await registerEscort({ city: '北京', nickname: '陪诊师小张' });
    expect(requestSpy).toHaveBeenCalledWith({
      url: '/api/v1/escorts',
      method: 'POST',
      data: { city: '北京', nickname: '陪诊师小张' },
      baseURL: 'http://127.0.0.1:8089',
    });
  });

  it('updateLocation PATCHes full location', async () => {
    await updateLocation({ city: '北京', lat: 39.9, lng: 116.4, address: 'xx' });
    expect(requestSpy.mock.calls[0]![0].method).toBe('PATCH');
    expect(requestSpy.mock.calls[0]![0].url).toBe('/api/v1/escorts/me/location');
  });

  it('updateCity PATCHes city', async () => {
    await updateCity('上海');
    expect(requestSpy.mock.calls[0]![0]).toEqual({
      url: '/api/v1/escorts/me/city',
      method: 'PATCH',
      data: { city: '上海' },
      baseURL: 'http://127.0.0.1:8089',
    });
  });
});

describe('api/escort · qualifications', () => {
  it('createQualification POSTs body', async () => {
    await createQualification({ type: 'medical', title: '医师资格证', cert_url: 'https://...' });
    expect(requestSpy.mock.calls[0]![0].url).toBe('/api/v1/escorts/qualifications');
  });

  it('updateQualification PATCHes body', async () => {
    await updateQualification(7, { title: '新标题' });
    expect(requestSpy.mock.calls[0]![0]).toEqual({
      url: '/api/v1/escorts/qualifications/7',
      method: 'PATCH',
      data: { title: '新标题' },
      baseURL: 'http://127.0.0.1:8089',
    });
  });

  it('deleteQualification DELETEs', async () => {
    await deleteQualification(7);
    expect(requestSpy.mock.calls[0]![0].method).toBe('DELETE');
    expect(requestSpy.mock.calls[0]![0].url).toBe('/api/v1/escorts/qualifications/7');
  });
});

describe('api/escort · availabilities', () => {
  it('addAvailability PUTs slot', async () => {
    await addAvailability({ start_at: '2026-10-01T09:00:00Z', end_at: '2026-10-01T18:00:00Z', remark: '全天' });
    expect(requestSpy.mock.calls[0]![0].method).toBe('PUT');
    expect(requestSpy.mock.calls[0]![0].url).toBe('/api/v1/escorts/me/availabilities');
  });

  it('removeAvailability DELETEs by id', async () => {
    await removeAvailability(7);
    expect(requestSpy.mock.calls[0]![0]).toEqual({
      url: '/api/v1/escorts/me/availabilities/7',
      method: 'DELETE',
      baseURL: 'http://127.0.0.1:8089',
    });
  });
});