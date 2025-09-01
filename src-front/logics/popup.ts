import { useModal, useVfm } from 'vue-final-modal'
import MyBotModal from '@/modals/MyBotModal.vue';
import PythonModal from '@/modals/PythonModal.vue';
import ContextModal from '@/modals/ContextModal.vue';
import UseToolModal from '@/modals/UseToolModal.vue';
import BrowserModal from '@/modals/BrowserModal.vue';
import DatabaseModal from '@/modals/DatabaseModal.vue';
import ModelProvider from '@/modals/ModelProvider.vue';
import McpConfigModal from '@/modals/McpConfigModal.vue';
import WelcomeModal from '@/modals/WelcomeModal.vue';

export const showContext = (context: any) => {
  const theContextView = useModal({
    component: ContextModal,
    attrs: {
      onSubmit: () => {
        theContextView.close()
      },
      onCancel: () => {
        theContextView.close()
      },
    },
  })

  var attrs = theContextView.options.attrs || {}
  Object.assign(attrs, { context })
  theContextView.open()
}

export const showMyBotForm = (info: any) => {
  const myBotView = useModal({
    component: MyBotModal,
    attrs: {
      onSubmit: () => {
        myBotView.close()
      },
      onCancel: () => {
        myBotView.close()
      },
    },
  })

  var attrs = myBotView.options.attrs || {}
  Object.assign(attrs, { model: info })
  myBotView.open()
}

export const showUseToolModal = (info: any) => {
  const useToolModal = useModal({
    component: UseToolModal,
    attrs: {
      tool: info,
      onSubmit: () => {
        useToolModal.close()
      },
      onCancel: () => {
        useToolModal.close()
      },
    },
  })
  useToolModal.open()
}

export const showSetMcpModal = (info: McpServer, callback: CallableFunction) => {
  if (useVfm().get('theMcpConfigModal')) {
    return useVfm().open('theMcpConfigModal')
  }
  const theMcpConfigModal = useModal({
    component: McpConfigModal,
    attrs: {
      onSubmit: (data: McpServer) => {
        theMcpConfigModal.close()
        callback && callback(data, 'submit')
      },
      onDelete: () => {
        theMcpConfigModal.close()
        callback && callback(null, 'delete')
      },
      onCancel: () => {
        theMcpConfigModal.close()
      },
    },
  })
  var attrs = theMcpConfigModal.options.attrs || {}
  Object.assign(attrs, { model: info })
  theMcpConfigModal.open()
}

export const showPythonModal = () => {
  if (useVfm().get('thePythonModal')) {
    return useVfm().open('thePythonModal')
  }
  const thePythonModal = useModal({
    component: PythonModal,
    attrs: {
      onSubmit: () => {
        thePythonModal.close()
      },
      onCancel: () => {
        thePythonModal.close()
      },
    },
  })

  thePythonModal.open()
}

export const showDatabaseModal = () => {
  if (useVfm().get('theDatabaseModal')) {
    return
  }

  const theDatabaseModal = useModal({
    component: DatabaseModal,
    attrs: {
      onSubmit: () => {
        theDatabaseModal.close()
      },
      onCancel: () => {
        theDatabaseModal.close()
      },
    },
  })

  theDatabaseModal.open()
}

export const showBrowserModal = () => {
  if (useVfm().get('theBrowserModal')) {
    return
  }
  const theBrowserModal = useModal({
    component: BrowserModal,
    attrs: {
      onSubmit: () => {
        theBrowserModal.close()
      },
      onCancel: () => {
        theBrowserModal.close()
      },
    },
  })
  theBrowserModal.open()
}

export const showUseModelPopup = (gateway: string) => {
  if (useVfm().get('theModelProvider')) {
    return
  }
  const theModelProvider = useModal({
    component: ModelProvider,
    attrs: {
      onSubmit: () => {
        theModelProvider.close()
      },
      onCancel: () => {
        theModelProvider.close()
      },
    },
  })
  var attrs = theModelProvider.options.attrs || {}
  Object.assign(attrs, { gateway, from: 'model' })
  theModelProvider.open()
}

export const showProviderPopup = (provider = '', callback: CallableFunction) => {
  if (useVfm().get('theModelProvider')) {
    return
  }
  const theModelProvider = useModal({
    component: ModelProvider,
    attrs: {
      onSubmit: (data) => {
        theModelProvider.close()
        callback && callback(data)
      },
      onCancel: () => {
        theModelProvider.close()
      },
    },
  })
  var attrs = theModelProvider.options.attrs || {}
  Object.assign(attrs, { provider, from: 'provider' })
  theModelProvider.open()
}

export const showWelcomeModal = (epigraph: string) => {
  if (useVfm().get('theWelcomeModal')) {
    return
  }
  const theWelcomeModal = useModal({
    component: WelcomeModal,
    attrs: {
      onSubmit: () => {
        theWelcomeModal.close()
      },
      onCancel: () => {
        theWelcomeModal.close()
      },
    },
  })
  var attrs = theWelcomeModal.options.attrs || {}
  Object.assign(attrs, { epigraph })
  theWelcomeModal.open()
}