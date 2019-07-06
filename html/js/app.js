window.addEventListener("load", () => {
    window.vue = new Vue({
        el: "#app",
        data: {
            message: "merhaba dünya",
            headerHtml: "",
            m: "boş",
        },
        created() {
            fetch("/html/genel/header.html")
                .then((res) => {
                    return res.text()
                })
                .then((res) => {
                    this.headerHtml = res;
                });
            this.m = get("m");
        }
    });
});

function get(variable) {
    let query = window.location.search.substring(1);
    let vars = query.split("&");
    for (let i = 0; i < vars.length; i++) {
        let pair = vars[i].split("=");
        if (pair[0] == variable) {
            return pair[1];
        }
    }
    return (false);
}
