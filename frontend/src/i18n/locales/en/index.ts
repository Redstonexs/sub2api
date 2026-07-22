import landing from './landing'
import common from './common'
import dashboard from './dashboard'
import batchImage from './batchImage'
import admin from './admin'
import misc from './misc'
import fork from './fork'
import { deepMerge } from '../../deepMerge'

export default deepMerge(
  {
    ...landing,
    ...common,
    ...dashboard,
    ...batchImage,
    admin,
    ...misc,
  },
  fork
)
