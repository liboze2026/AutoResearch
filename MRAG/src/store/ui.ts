import { defineStore } from "pinia";

export const useUiStore = defineStore("ui", {
  state: () => ({
    collapsed: false
  }),
  actions: {
    toggleSidebar() {
      this.collapsed = !this.collapsed;
    }
  }
});
