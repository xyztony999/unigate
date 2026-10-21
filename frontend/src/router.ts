import { createRouter, createWebHistory } from 'vue-router'
import { contract } from './lib/contract'
const LoginView=()=>import('./views/LoginView.vue')
const ScannerView=()=>import('./views/ScannerView.vue')
const ParticipantView=()=>import('./views/ParticipantView.vue')
const WorkspaceView=()=>import('./views/WorkspaceView.vue')
export const router=createRouter({history:createWebHistory(),routes:[
  {path:'/',redirect:'/manage/events'},
  ...contract.pages.map(page=>({path:page.route,name:page.id,component:page.id==='S01'?LoginView:['O09','C01','P04'].includes(page.id)?ScannerView:['P01','P02','P03','S04'].includes(page.id)?ParticipantView:WorkspaceView,meta:{page}})),
  {path:'/:pathMatch(.*)*',redirect:'/'}
]})
