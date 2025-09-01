
import { request } from "./index"

export const getGlobalInfo = () => {
  return request.get('/global')
}