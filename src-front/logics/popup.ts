import { useModal, useVfm } from 'vue-final-modal'
import BotInfoModal from '@/modals/BotInfoModal.vue';
import WelcomeModal from '@/modals/WelcomeModal.vue';
import ContextModal from '@/modals/ContextModal.vue';
import UseToolModal from '@/modals/UseToolModal.vue';
import ProviderModal from '@/modals/ProviderModal.vue';
import SetupEnvModal from '@/modals/SetupEnvModal.vue';
import McpConfigModal from '@/modals/McpConfigModal.vue';

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

export const showBotInfoForm = (info: any) => {
  const theBotView = useModal({
    component: BotInfoModal,
    attrs: {
      onSubmit: () => {
        theBotView.close()
      },
      onCancel: () => {
        theBotView.close()
      },
    },
  })

  var attrs = theBotView.options.attrs || {}
  Object.assign(attrs, { model: info })
  theBotView.open()
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

export const showSetupEnvModal = () => {
  if (useVfm().get('theSetupEnvModal')) {
    return useVfm().open('theSetupEnvModal')
  }
  const theSetupEnvModal = useModal({
    component: SetupEnvModal,
    attrs: {
      onCancel: () => {
        theSetupEnvModal.close()
      },
    },
  })

  theSetupEnvModal.open()
}

export const showUseModelPopup = () => {
  if (useVfm().get('theProviderModal')) {
    return
  }
  const theProviderModal = useModal({
    component: ProviderModal,
    attrs: {
      onSubmit: () => {
        theProviderModal.close()
      },
      onCancel: () => {
        theProviderModal.close()
      },
    },
  })
  var attrs = theProviderModal.options.attrs || {}
  Object.assign(attrs, { from: 'model' })
  theProviderModal.open()
}

export const showProviderPopup = (provider = '', callback: CallableFunction) => {
  if (useVfm().get('theProviderModal')) {
    return
  }
  const theProviderModal = useModal({
    component: ProviderModal,
    attrs: {
      onSubmit: (data) => {
        theProviderModal.close()
        callback && callback(data)
      },
      onCancel: () => {
        theProviderModal.close()
      },
    },
  })
  var attrs = theProviderModal.options.attrs || {}
  Object.assign(attrs, { provider, from: 'provider' })
  theProviderModal.open()
}

export const showWelcomeModal = (gateway: string, initialState = {}) => {
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
  Object.assign(attrs, { gateway, initialState })
  theWelcomeModal.open()
}