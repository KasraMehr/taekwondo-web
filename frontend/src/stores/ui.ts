import { defineStore } from "pinia";
import { ref } from "vue";

export type Page = "home" | "tournament" | "league" | "teamTournament" | "poomsae";
export type PoomsaeTab = 'athletes' | 'draw' | 'scores' | 'standings' | 'print' | 'settings'

export type TournamentTab =
    | "athletes"
    | "weighIn"
    | "draw"
    | "matches"
    | "standings"
    | "export"
    | "settings";
export type LeagueTab = "setup" | "roster" | "overview" | "weeks" | "settings"

export type TeamTournamentTab =
    | "teams"
    | "lineups"
    | "draw"
    | "matches"
    | "standings"
    | "export";

export const useUiStore = defineStore("ui", () => {
    const page = ref<Page>("home");
    const homeSection = ref<'tournaments' | 'leagues' | 'poomsae'>('tournaments');
    const poomsaeEventId = ref('');
    const poomsaeDivisionId = ref('');
    const poomsaeTab = ref<PoomsaeTab>('athletes');
    const tab = ref<TournamentTab>("athletes");
    const leagueTab = ref<LeagueTab>("setup");
    const teamTab = ref<TeamTournamentTab>("teams");

    function goHome() {
        page.value = "home";
        tab.value = "athletes";
        leagueTab.value = "setup";
        teamTab.value = "teams";
    }

    function goTournament() {
        page.value = "tournament";
    }

    function goLeagueDetail() {
        page.value = "league";
        leagueTab.value = "setup";
    }

    function goTeamTournament() {
        page.value = "teamTournament";
        teamTab.value = "teams";
    }

    return {
        page,
        homeSection, poomsaeEventId, poomsaeDivisionId, poomsaeTab,
        tab,
        leagueTab,
        teamTab,
        goHome,
        goTournament,
        goLeagueDetail,
        goTeamTournament,
    };
});
